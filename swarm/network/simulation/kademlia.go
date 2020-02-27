// Authored and revised by YOC team, 2018
// License placeholder #1

package simulation

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/p2p/discover"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/swarm/network"
)

// BucketKeyKademlia is the key to be used for storing the kademlia
// instance for particuar node, usually inside the ServiceFunc function.
var BucketKeyKademlia BucketKey = "kademlia"

// WaitTillHealthy is blocking until the health of all kademlias is true.
// If error is not nil, a map of kademlia that was found not healthy is returned.
func (s *Simulation) WaitTillHealthy(ctx context.Context, kadMinProxSize int) (ill map[discover.NodeID]*network.Kademlia, err error) {
	// Prepare PeerPot map for checking Kademlia health
	var ppmap map[string]*network.PeerPot
	kademlias := s.kademlias()
	addrs := make([][]byte, 0, len(kademlias))
	for _, k := range kademlias {
		addrs = append(addrs, k.BaseAddr())
	}
	ppmap = network.NewPeerPotMap(kadMinProxSize, addrs)

	// Wait for healthy Kademlia on every node before checking files
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	ill = make(map[discover.NodeID]*network.Kademlia)
	for {
		select {
		case <-ctx.Done():
			return ill, ctx.Err()
		case <-ticker.C:
			for k := range ill {
				delete(ill, k)
			}
			log.Debug("kademlia health check", "addr count", len(addrs))
			for id, k := range kademlias {
				//PeerPot for this node
				addr := common.Bytes2Hex(k.BaseAddr())
				pp := ppmap[addr]
				//call Healthy RPC
				h := k.Healthy(pp)
				//print info
				log.Debug(k.String())
				log.Debug("kademlia", "empty bins", pp.EmptyBins, "gotNN", h.GotNN, "knowNN", h.KnowNN, "full", h.Full)
				log.Debug("kademlia", "health", h.GotNN && h.KnowNN && h.Full, "addr", hex.EncodeToString(k.BaseAddr()), "node", id)
				log.Debug("kademlia", "ill condition", !h.GotNN || !h.Full, "addr", hex.EncodeToString(k.BaseAddr()), "node", id)
				if !h.GotNN || !h.Full {
					ill[id] = k
				}
			}
			if len(ill) == 0 {
				return nil, nil
			}
		}
	}
}

// kademlias returns all Kademlia instances that are set
// in simulation bucket.
func (s *Simulation) kademlias() (ks map[discover.NodeID]*network.Kademlia) {
	items := s.UpNodesItems(BucketKeyKademlia)
	ks = make(map[discover.NodeID]*network.Kademlia, len(items))
	for id, v := range items {
		k, ok := v.(*network.Kademlia)
		if !ok {
			continue
		}
		ks[id] = k
	}
	return ks
}
