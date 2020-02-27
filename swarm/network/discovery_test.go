// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package network

import (
	"testing"

	p2ptest "github.com/Yocoin15/Yocoin_Sources/p2p/testing"
)

/***
 *
 * - after connect, that outgoing subpeersmsg is sent
 *
 */
func TestDiscovery(t *testing.T) {
	params := NewHiveParams()
	s, pp := newHiveTester(t, params, 1, nil)

	id := s.IDs[0]
	raddr := NewAddrFromNodeID(id)
	pp.Register([]OverlayAddr{OverlayAddr(raddr)})

	// start the hive and wait for the connection
	pp.Start(s.Server)
	defer pp.Stop()

	// send subPeersMsg to the peer
	err := s.TestExchanges(p2ptest.Exchange{
		Label: "outgoing subPeersMsg",
		Expects: []p2ptest.Expect{
			{
				Code: 1,
				Msg:  &subPeersMsg{Depth: 0},
				Peer: id,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}
}
