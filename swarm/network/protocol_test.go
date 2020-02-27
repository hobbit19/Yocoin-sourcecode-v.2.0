// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package network

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
	"github.com/Yocoin15/Yocoin_Sources/p2p/discover"
	"github.com/Yocoin15/Yocoin_Sources/p2p/protocols"
	p2ptest "github.com/Yocoin15/Yocoin_Sources/p2p/testing"
)

var (
	loglevel = flag.Int("loglevel", 2, "verbosity of logs")
)

func init() {
	flag.Parse()
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
}

type testStore struct {
	sync.Mutex

	values map[string][]byte
}

func newTestStore() *testStore {
	return &testStore{values: make(map[string][]byte)}
}

func (t *testStore) Load(key string) ([]byte, error) {
	t.Lock()
	defer t.Unlock()
	v, ok := t.values[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

func (t *testStore) Save(key string, v []byte) error {
	t.Lock()
	defer t.Unlock()
	t.values[key] = v
	return nil
}

func HandshakeMsgExchange(lhs, rhs *HandshakeMsg, id discover.NodeID) []p2ptest.Exchange {

	return []p2ptest.Exchange{
		{
			Expects: []p2ptest.Expect{
				{
					Code: 0,
					Msg:  lhs,
					Peer: id,
				},
			},
		},
		{
			Triggers: []p2ptest.Trigger{
				{
					Code: 0,
					Msg:  rhs,
					Peer: id,
				},
			},
		},
	}
}

func newBzzBaseTester(t *testing.T, n int, addr *BzzAddr, spec *protocols.Spec, run func(*BzzPeer) error) *bzzTester {
	cs := make(map[string]chan bool)

	srv := func(p *BzzPeer) error {
		defer func() {
			if cs[p.ID().String()] != nil {
				close(cs[p.ID().String()])
			}
		}()
		return run(p)
	}

	protocol := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		return srv(&BzzPeer{
			Peer:      protocols.NewPeer(p, rw, spec),
			localAddr: addr,
			BzzAddr:   NewAddrFromNodeID(p.ID()),
		})
	}

	s := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), n, protocol)

	for _, id := range s.IDs {
		cs[id.String()] = make(chan bool)
	}

	return &bzzTester{
		addr:           addr,
		ProtocolTester: s,
		cs:             cs,
	}
}

type bzzTester struct {
	*p2ptest.ProtocolTester
	addr *BzzAddr
	cs   map[string]chan bool
}

func newBzzHandshakeTester(t *testing.T, n int, addr *BzzAddr) *bzzTester {
	config := &BzzConfig{
		OverlayAddr:  addr.Over(),
		UnderlayAddr: addr.Under(),
		HiveParams:   NewHiveParams(),
		NetworkID:    DefaultNetworkID,
	}
	kad := NewKademlia(addr.OAddr, NewKadParams())
	bzz := NewBzz(config, kad, nil, nil, nil)

	s := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), 1, bzz.runBzz)

	return &bzzTester{
		addr:           addr,
		ProtocolTester: s,
	}
}

// should test handshakes in one exchange? parallelisation
func (s *bzzTester) testHandshake(lhs, rhs *HandshakeMsg, disconnects ...*p2ptest.Disconnect) error {
	var peers []discover.NodeID
	id := NewNodeIDFromAddr(rhs.Addr)
	if len(disconnects) > 0 {
		for _, d := range disconnects {
			peers = append(peers, d.Peer)
		}
	} else {
		peers = []discover.NodeID{id}
	}

	if err := s.TestExchanges(HandshakeMsgExchange(lhs, rhs, id)...); err != nil {
		return err
	}

	if len(disconnects) > 0 {
		return s.TestDisconnected(disconnects...)
	}

	// If we don't expect disconnect, ensure peers remain connected
	err := s.TestDisconnected(&p2ptest.Disconnect{
		Peer:  s.IDs[0],
		Error: nil,
	})

	if err == nil {
		return fmt.Errorf("Unexpected peer disconnect")
	}

	if err.Error() != "timed out waiting for peers to disconnect" {
		return err
	}

	return nil
}

func correctBzzHandshake(addr *BzzAddr) *HandshakeMsg {
	return &HandshakeMsg{
		Version:   5,
		NetworkID: DefaultNetworkID,
		Addr:      addr,
	}
}

func TestBzzHandshakeNetworkIDMismatch(t *testing.T) {
	addr := RandomAddr()
	s := newBzzHandshakeTester(t, 1, addr)
	id := s.IDs[0]

	err := s.testHandshake(
		correctBzzHandshake(addr),
		&HandshakeMsg{Version: 5, NetworkID: 321, Addr: NewAddrFromNodeID(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("Handshake error: Message handler error: (msg code 0): network id mismatch 321 (!= 3)")},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestBzzHandshakeVersionMismatch(t *testing.T) {
	addr := RandomAddr()
	s := newBzzHandshakeTester(t, 1, addr)
	id := s.IDs[0]

	err := s.testHandshake(
		correctBzzHandshake(addr),
		&HandshakeMsg{Version: 0, NetworkID: 3, Addr: NewAddrFromNodeID(id)},
		&p2ptest.Disconnect{Peer: id, Error: fmt.Errorf("Handshake error: Message handler error: (msg code 0): version mismatch 0 (!= 5)")},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestBzzHandshakeSuccess(t *testing.T) {
	addr := RandomAddr()
	s := newBzzHandshakeTester(t, 1, addr)
	id := s.IDs[0]

	err := s.testHandshake(
		correctBzzHandshake(addr),
		&HandshakeMsg{Version: 5, NetworkID: 3, Addr: NewAddrFromNodeID(id)},
	)

	if err != nil {
		t.Fatal(err)
	}
}
