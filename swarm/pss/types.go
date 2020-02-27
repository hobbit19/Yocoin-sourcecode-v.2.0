// Authored and revised by YOC team, 2018
// License placeholder #1

package pss

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/hexutil"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
	"github.com/Yocoin15/Yocoin_Sources/rlp"
	"github.com/Yocoin15/Yocoin_Sources/swarm/storage"
	whisper "github.com/Yocoin15/Yocoin_Sources/whisper/whisperv5"
)

const (
	defaultWhisperTTL = 6000
)

const (
	pssControlSym = 1
	pssControlRaw = 1 << 1
)

var (
	topicHashMutex = sync.Mutex{}
	topicHashFunc  = storage.MakeHashFunc("SHA256")()
	rawTopic       = Topic{}
)

// Topic is the PSS encapsulation of the Whisper topic type
type Topic whisper.TopicType

func (t *Topic) String() string {
	return hexutil.Encode(t[:])
}

// MarshalJSON implements the json.Marshaler interface
func (t Topic) MarshalJSON() (b []byte, err error) {
	return json.Marshal(t.String())
}

// MarshalJSON implements the json.Marshaler interface
func (t *Topic) UnmarshalJSON(input []byte) error {
	topicbytes, err := hexutil.Decode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	copy(t[:], topicbytes)
	return nil
}

// PssAddress is an alias for []byte. It represents a variable length address
type PssAddress []byte

// MarshalJSON implements the json.Marshaler interface
func (a PssAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Encode(a[:]))
}

// UnmarshalJSON implements the json.Marshaler interface
func (a *PssAddress) UnmarshalJSON(input []byte) error {
	b, err := hexutil.Decode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	for _, bb := range b {
		*a = append(*a, bb)
	}
	return nil
}

// holds the digest of a message used for caching
type pssDigest [digestLength]byte

// conceals bitwise operations on the control flags byte
type msgParams struct {
	raw bool
	sym bool
}

func newMsgParamsFromBytes(paramBytes []byte) *msgParams {
	if len(paramBytes) != 1 {
		return nil
	}
	return &msgParams{
		raw: paramBytes[0]&pssControlRaw > 0,
		sym: paramBytes[0]&pssControlSym > 0,
	}
}

func (m *msgParams) Bytes() (paramBytes []byte) {
	var b byte
	if m.raw {
		b |= pssControlRaw
	}
	if m.sym {
		b |= pssControlSym
	}
	paramBytes = append(paramBytes, b)
	return paramBytes
}

// PssMsg encapsulates messages transported over pss.
type PssMsg struct {
	To      []byte
	Control []byte
	Expire  uint32
	Payload *whisper.Envelope
}

func newPssMsg(param *msgParams) *PssMsg {
	return &PssMsg{
		Control: param.Bytes(),
	}
}

// message is flagged as raw / external encryption
func (msg *PssMsg) isRaw() bool {
	return msg.Control[0]&pssControlRaw > 0
}

// message is flagged as symmetrically encrypted
func (msg *PssMsg) isSym() bool {
	return msg.Control[0]&pssControlSym > 0
}

// serializes the message for use in cache
func (msg *PssMsg) serialize() []byte {
	rlpdata, _ := rlp.EncodeToBytes(struct {
		To      []byte
		Payload *whisper.Envelope
	}{
		To:      msg.To,
		Payload: msg.Payload,
	})
	return rlpdata
}

// String representation of PssMsg
func (msg *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %x", common.ToHex(msg.To))
}

// Signature for a message handler function for a PssMsg
//
// Implementations of this type are passed to Pss.Register together with a topic,
type Handler func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error

// the stateStore handles saving and loading PSS peers and their corresponding keys
// it is currently unimplemented
type stateStore struct {
	values map[string][]byte
}

func newStateStore() *stateStore {
	return &stateStore{values: make(map[string][]byte)}
}

func (store *stateStore) Load(key string) ([]byte, error) {
	return nil, nil
}

func (store *stateStore) Save(key string, v []byte) error {
	return nil
}

// BytesToTopic hashes an arbitrary length byte slice and truncates it to the length of a topic, using only the first bytes of the digest
func BytesToTopic(b []byte) Topic {
	topicHashMutex.Lock()
	defer topicHashMutex.Unlock()
	topicHashFunc.Reset()
	topicHashFunc.Write(b)
	return Topic(whisper.BytesToTopic(topicHashFunc.Sum(nil)))
}
