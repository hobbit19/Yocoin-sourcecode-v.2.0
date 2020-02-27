// Authored and revised by YOC team, 2018
// License placeholder #1

package network

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/node"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
	"github.com/Yocoin15/Yocoin_Sources/p2p/discover"
	"github.com/Yocoin15/Yocoin_Sources/p2p/simulations"
	"github.com/Yocoin15/Yocoin_Sources/p2p/simulations/adapters"
	"github.com/Yocoin15/Yocoin_Sources/rpc"
)

var (
	currentNetworkID int
	cnt              int
	nodeMap          map[int][]discover.NodeID
	kademlias        map[discover.NodeID]*Kademlia
)

const (
	NumberOfNets = 4
	MaxTimeout   = 6
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().Unix())
}

/*
Run the network ID test.
The test creates one simulations.Network instance,
a number of nodes, then connects nodes with each other in this network.

Each node gets a network ID assigned according to the number of networks.
Having more network IDs is just arbitrary in order to exclude
false positives.

Nodes should only connect with other nodes with the same network ID.
After the setup phase, the test checks on each node if it has the
expected node connections (excluding those not sharing the network ID).
*/
func TestNetworkID(t *testing.T) {
	log.Debug("Start test")
	//arbitrarily set the number of nodes. It could be any number
	numNodes := 24
	//the nodeMap maps all nodes (slice value) with the same network ID (key)
	nodeMap = make(map[int][]discover.NodeID)
	//set up the network and connect nodes
	net, err := setupNetwork(numNodes)
	if err != nil {
		t.Fatalf("Error setting up network: %v", err)
	}
	defer func() {
		//shutdown the snapshot network
		log.Trace("Shutting down network")
		net.Shutdown()
	}()
	//let's sleep to ensure all nodes are connected
	time.Sleep(1 * time.Second)
	//for each group sharing the same network ID...
	for _, netIDGroup := range nodeMap {
		log.Trace("netIDGroup size", "size", len(netIDGroup))
		//...check that their size of the kademlia is of the expected size
		//the assumption is that it should be the size of the group minus 1 (the node itself)
		for _, node := range netIDGroup {
			if kademlias[node].addrs.Size() != len(netIDGroup)-1 {
				t.Fatalf("Kademlia size has not expected peer size. Kademlia size: %d, expected size: %d", kademlias[node].addrs.Size(), len(netIDGroup)-1)
			}
			kademlias[node].EachAddr(nil, 0, func(addr OverlayAddr, _ int, _ bool) bool {
				found := false
				for _, nd := range netIDGroup {
					p := ToOverlayAddr(nd.Bytes())
					if bytes.Equal(p, addr.Address()) {
						found = true
					}
				}
				if !found {
					t.Fatalf("Expected node not found for node %s", node.String())
				}
				return true
			})
		}
	}
	log.Info("Test terminated successfully")
}

// setup simulated network with bzz/discovery and pss services.
// connects nodes in a circle
// if allowRaw is set, omission of builtin pss encryption is enabled (see PssParams)
func setupNetwork(numnodes int) (net *simulations.Network, err error) {
	log.Debug("Setting up network")
	quitC := make(chan struct{})
	errc := make(chan error)
	nodes := make([]*simulations.Node, numnodes)
	if numnodes < 16 {
		return nil, fmt.Errorf("Minimum sixteen nodes in network")
	}
	adapter := adapters.NewSimAdapter(newServices())
	//create the network
	net = simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "NetworkIdTestNet",
		DefaultService: "bzz",
	})
	log.Debug("Creating networks and nodes")

	var connCount int

	//create nodes and connect them to each other
	for i := 0; i < numnodes; i++ {
		log.Trace("iteration: ", "i", i)
		nodeconf := adapters.RandomNodeConfig()
		nodes[i], err = net.NewNodeWithConfig(nodeconf)
		if err != nil {
			return nil, fmt.Errorf("error creating node %d: %v", i, err)
		}
		err = net.Start(nodes[i].ID())
		if err != nil {
			return nil, fmt.Errorf("error starting node %d: %v", i, err)
		}
		client, err := nodes[i].Client()
		if err != nil {
			return nil, fmt.Errorf("create node %d rpc client fail: %v", i, err)
		}
		//now setup and start event watching in order to know when we can upload
		ctx, watchCancel := context.WithTimeout(context.Background(), MaxTimeout*time.Second)
		defer watchCancel()
		watchSubscriptionEvents(ctx, nodes[i].ID(), client, errc, quitC)
		//on every iteration we connect to all previous ones
		for k := i - 1; k >= 0; k-- {
			connCount++
			log.Debug(fmt.Sprintf("Connecting node %d with node %d; connection count is %d", i, k, connCount))
			err = net.Connect(nodes[i].ID(), nodes[k].ID())
			if err != nil {
				if !strings.Contains(err.Error(), "already connected") {
					return nil, fmt.Errorf("error connecting nodes: %v", err)
				}
			}
		}
	}
	//now wait until the number of expected subscriptions has been finished
	//`watchSubscriptionEvents` will write with a `nil` value to errc
	for err := range errc {
		if err != nil {
			return nil, err
		}
		//`nil` received, decrement count
		connCount--
		log.Trace("count down", "cnt", connCount)
		//all subscriptions received
		if connCount == 0 {
			close(quitC)
			break
		}
	}
	log.Debug("Network setup phase terminated")
	return net, nil
}

func newServices() adapters.Services {
	kademlias = make(map[discover.NodeID]*Kademlia)
	kademlia := func(id discover.NodeID) *Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		addr := NewAddrFromNodeID(id)
		params := NewKadParams()
		params.MinProxBinSize = 2
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = NewKademlia(addr.Over(), params)
		return kademlias[id]
	}
	return adapters.Services{
		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
			addr := NewAddrFromNodeID(ctx.Config.ID)
			hp := NewHiveParams()
			hp.Discovery = false
			cnt++
			//assign the network ID
			currentNetworkID = cnt % NumberOfNets
			if ok := nodeMap[currentNetworkID]; ok == nil {
				nodeMap[currentNetworkID] = make([]discover.NodeID, 0)
			}
			//add this node to the group sharing the same network ID
			nodeMap[currentNetworkID] = append(nodeMap[currentNetworkID], ctx.Config.ID)
			log.Debug("current network ID:", "id", currentNetworkID)
			config := &BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
				NetworkID:    uint64(currentNetworkID),
			}
			return NewBzz(config, kademlia(ctx.Config.ID), nil, nil, nil), nil
		},
	}
}

func watchSubscriptionEvents(ctx context.Context, id discover.NodeID, client *rpc.Client, errc chan error, quitC chan struct{}) {
	events := make(chan *p2p.PeerEvent)
	sub, err := client.Subscribe(context.Background(), "admin", events, "peerEvents")
	if err != nil {
		log.Error(err.Error())
		errc <- fmt.Errorf("error getting peer events for node %v: %s", id, err)
		return
	}
	go func() {
		defer func() {
			sub.Unsubscribe()
			log.Trace("watch subscription events: unsubscribe", "id", id)
		}()

		for {
			select {
			case <-quitC:
				return
			case <-ctx.Done():
				select {
				case errc <- ctx.Err():
				case <-quitC:
				}
				return
			case e := <-events:
				if e.Type == p2p.PeerEventTypeAdd {
					errc <- nil
				}
			case err := <-sub.Err():
				if err != nil {
					select {
					case errc <- fmt.Errorf("error getting peer events for node %v: %v", id, err):
					case <-quitC:
					}
					return
				}
			}
		}
	}()
}
