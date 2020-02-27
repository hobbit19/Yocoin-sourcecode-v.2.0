// Authored and revised by YOC team, 2016-2018
// License placeholder #1

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package yocoin

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Yocoin15/Yocoin_Sources/core"
	"github.com/Yocoin15/Yocoin_Sources/internal/debug"
	"github.com/Yocoin15/Yocoin_Sources/les"
	"github.com/Yocoin15/Yocoin_Sources/node"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
	"github.com/Yocoin15/Yocoin_Sources/p2p/nat"
	"github.com/Yocoin15/Yocoin_Sources/params"
	whisper "github.com/Yocoin15/Yocoin_Sources/whisper/whisperv6"
	"github.com/Yocoin15/Yocoin_Sources/yoc"
	"github.com/Yocoin15/Yocoin_Sources/yoc/downloader"
	"github.com/Yocoin15/Yocoin_Sources/yocclient"
	"github.com/Yocoin15/Yocoin_Sources/yocstats"
)

// NodeConfig represents the collection of configuration values to fine tune the Yocoin
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by yocoin to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// YoCoinEnabled specifies whether the node should run the YoCoin protocol.
	YoCoinEnabled bool

	// YoCoinNetworkID is the network identifier used by the YoCoin protocol to
	// decide if remote peers should be accepted or not.
	YoCoinNetworkID int64 // uint64 in truth, but Java can't handle that...

	// YoCoinGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	YoCoinGenesis string

	// YoCoinDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	YoCoinDatabaseCache int

	// YoCoinNetStats is a netstats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	YoCoinNetStats string

	// WhisperEnabled specifies whether the node should run the Whisper protocol.
	WhisperEnabled bool

	// Listening address of pprof server.
	PprofAddress string
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:      FoundationBootnodes(),
	MaxPeers:            25,
	YoCoinEnabled:       true,
	YoCoinNetworkID:     1,
	YoCoinDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Yocoin YoCoin node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Yocoin node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}

	if config.PprofAddress != "" {
		debug.StartPProf(config.PprofAddress)
	}

	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.VersionWithMeta,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	debug.Memsize.Add("node", rawStack)

	var genesis *core.Genesis
	if config.YoCoinGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.YoCoinGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.YoCoinGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.YoCoinNetworkID == 1 {
				config.YoCoinNetworkID = 3
			}
		}
	}
	// Register the YoCoin protocol if requested
	if config.YoCoinEnabled {
		yocConf := yoc.DefaultConfig
		yocConf.Genesis = genesis
		yocConf.SyncMode = downloader.LightSync
		yocConf.NetworkId = uint64(config.YoCoinNetworkID)
		yocConf.DatabaseCache = config.YoCoinDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &yocConf)
		}); err != nil {
			return nil, fmt.Errorf("ethereum init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.YoCoinNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightYoCoin
				ctx.Service(&lesServ)

				return yocstats.New(config.YoCoinNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. If the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetYoCoinClient retrieves a client to access the YoCoin subsystem.
func (n *Node) GetYoCoinClient() (client *YoCoinClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &YoCoinClient{yocclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
