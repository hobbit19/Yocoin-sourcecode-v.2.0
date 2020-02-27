// Authored and revised by YOC team, 2016-2018
// License placeholder #1

// Package les implements the Light YoCoin Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/accounts"
	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/hexutil"
	"github.com/Yocoin15/Yocoin_Sources/consensus"
	"github.com/Yocoin15/Yocoin_Sources/core"
	"github.com/Yocoin15/Yocoin_Sources/core/bloombits"
	"github.com/Yocoin15/Yocoin_Sources/core/rawdb"
	"github.com/Yocoin15/Yocoin_Sources/core/types"
	"github.com/Yocoin15/Yocoin_Sources/event"
	"github.com/Yocoin15/Yocoin_Sources/internal/yocapi"
	"github.com/Yocoin15/Yocoin_Sources/light"
	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/node"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
	"github.com/Yocoin15/Yocoin_Sources/p2p/discv5"
	"github.com/Yocoin15/Yocoin_Sources/params"
	rpc "github.com/Yocoin15/Yocoin_Sources/rpc"
	"github.com/Yocoin15/Yocoin_Sources/yoc"
	"github.com/Yocoin15/Yocoin_Sources/yoc/downloader"
	"github.com/Yocoin15/Yocoin_Sources/yoc/filters"
	"github.com/Yocoin15/Yocoin_Sources/yoc/gasprice"
	"github.com/Yocoin15/Yocoin_Sources/yocdb"
)

type LightYoCoin struct {
	config *yoc.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb yocdb.Database // Block chain database

	bloomRequests                              chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *yocapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *yoc.Config) (*LightYoCoin, error) {
	chainDb, err := yoc.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lyoc := &LightYoCoin{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           yoc.CreateConsensusEngine(ctx, &config.Yochash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     yoc.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lyoc.relay = NewLesTxRelay(peers, lyoc.reqDist)
	lyoc.serverPool = newServerPool(chainDb, quitSync, &lyoc.wg)
	lyoc.retriever = newRetrieveManager(peers, lyoc.reqDist, lyoc.serverPool)
	lyoc.odr = NewLesOdr(chainDb, lyoc.chtIndexer, lyoc.bloomTrieIndexer, lyoc.bloomIndexer, lyoc.retriever)
	if lyoc.blockchain, err = light.NewLightChain(lyoc.odr, lyoc.chainConfig, lyoc.engine); err != nil {
		return nil, err
	}
	lyoc.bloomIndexer.Start(lyoc.blockchain)
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lyoc.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lyoc.txPool = light.NewTxPool(lyoc.chainConfig, lyoc.blockchain, lyoc.relay)
	if lyoc.protocolManager, err = NewProtocolManager(lyoc.chainConfig, true, ClientProtocolVersions, config.NetworkId, lyoc.eventMux, lyoc.engine, lyoc.peers, lyoc.blockchain, nil, chainDb, lyoc.odr, lyoc.relay, lyoc.serverPool, quitSync, &lyoc.wg); err != nil {
		return nil, err
	}
	lyoc.ApiBackend = &LesApiBackend{lyoc, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lyoc.ApiBackend.gpo = gasprice.NewOracle(lyoc.ApiBackend, gpoParams)
	return lyoc, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Yocbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Yocbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Yocbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the yocoin package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightYoCoin) APIs() []rpc.API {
	return append(yocapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "yoc",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "yoc",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "yoc",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightYoCoin) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightYoCoin) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightYoCoin) TxPool() *light.TxPool              { return s.txPool }
func (s *LightYoCoin) Engine() consensus.Engine           { return s.engine }
func (s *LightYoCoin) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightYoCoin) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightYoCoin) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightYoCoin) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// YoCoin protocol implementation.
func (s *LightYoCoin) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = yocapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// YoCoin protocol.
func (s *LightYoCoin) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.engine.Close()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
