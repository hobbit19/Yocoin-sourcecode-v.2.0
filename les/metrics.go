// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package les

import (
	"github.com/Yocoin15/Yocoin_Sources/metrics"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
)

var (
	/*	propTxnInPacketsMeter     = metrics.NewMeter("yoc/prop/txns/in/packets")
		propTxnInTrafficMeter     = metrics.NewMeter("yoc/prop/txns/in/traffic")
		propTxnOutPacketsMeter    = metrics.NewMeter("yoc/prop/txns/out/packets")
		propTxnOutTrafficMeter    = metrics.NewMeter("yoc/prop/txns/out/traffic")
		propHashInPacketsMeter    = metrics.NewMeter("yoc/prop/hashes/in/packets")
		propHashInTrafficMeter    = metrics.NewMeter("yoc/prop/hashes/in/traffic")
		propHashOutPacketsMeter   = metrics.NewMeter("yoc/prop/hashes/out/packets")
		propHashOutTrafficMeter   = metrics.NewMeter("yoc/prop/hashes/out/traffic")
		propBlockInPacketsMeter   = metrics.NewMeter("yoc/prop/blocks/in/packets")
		propBlockInTrafficMeter   = metrics.NewMeter("yoc/prop/blocks/in/traffic")
		propBlockOutPacketsMeter  = metrics.NewMeter("yoc/prop/blocks/out/packets")
		propBlockOutTrafficMeter  = metrics.NewMeter("yoc/prop/blocks/out/traffic")
		reqHashInPacketsMeter     = metrics.NewMeter("yoc/req/hashes/in/packets")
		reqHashInTrafficMeter     = metrics.NewMeter("yoc/req/hashes/in/traffic")
		reqHashOutPacketsMeter    = metrics.NewMeter("yoc/req/hashes/out/packets")
		reqHashOutTrafficMeter    = metrics.NewMeter("yoc/req/hashes/out/traffic")
		reqBlockInPacketsMeter    = metrics.NewMeter("yoc/req/blocks/in/packets")
		reqBlockInTrafficMeter    = metrics.NewMeter("yoc/req/blocks/in/traffic")
		reqBlockOutPacketsMeter   = metrics.NewMeter("yoc/req/blocks/out/packets")
		reqBlockOutTrafficMeter   = metrics.NewMeter("yoc/req/blocks/out/traffic")
		reqHeaderInPacketsMeter   = metrics.NewMeter("yoc/req/headers/in/packets")
		reqHeaderInTrafficMeter   = metrics.NewMeter("yoc/req/headers/in/traffic")
		reqHeaderOutPacketsMeter  = metrics.NewMeter("yoc/req/headers/out/packets")
		reqHeaderOutTrafficMeter  = metrics.NewMeter("yoc/req/headers/out/traffic")
		reqBodyInPacketsMeter     = metrics.NewMeter("yoc/req/bodies/in/packets")
		reqBodyInTrafficMeter     = metrics.NewMeter("yoc/req/bodies/in/traffic")
		reqBodyOutPacketsMeter    = metrics.NewMeter("yoc/req/bodies/out/packets")
		reqBodyOutTrafficMeter    = metrics.NewMeter("yoc/req/bodies/out/traffic")
		reqStateInPacketsMeter    = metrics.NewMeter("yoc/req/states/in/packets")
		reqStateInTrafficMeter    = metrics.NewMeter("yoc/req/states/in/traffic")
		reqStateOutPacketsMeter   = metrics.NewMeter("yoc/req/states/out/packets")
		reqStateOutTrafficMeter   = metrics.NewMeter("yoc/req/states/out/traffic")
		reqReceiptInPacketsMeter  = metrics.NewMeter("yoc/req/receipts/in/packets")
		reqReceiptInTrafficMeter  = metrics.NewMeter("yoc/req/receipts/in/traffic")
		reqReceiptOutPacketsMeter = metrics.NewMeter("yoc/req/receipts/out/packets")
		reqReceiptOutTrafficMeter = metrics.NewMeter("yoc/req/receipts/out/traffic")*/
	miscInPacketsMeter  = metrics.NewRegisteredMeter("les/misc/in/packets", nil)
	miscInTrafficMeter  = metrics.NewRegisteredMeter("les/misc/in/traffic", nil)
	miscOutPacketsMeter = metrics.NewRegisteredMeter("les/misc/out/packets", nil)
	miscOutTrafficMeter = metrics.NewRegisteredMeter("les/misc/out/traffic", nil)
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	if !metrics.Enabled {
		return rw
	}
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	// Read the message and short circuit in case of an error
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}
	// Account for the data traffic
	packets, traffic := miscInPacketsMeter, miscInTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	return msg, err
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	// Account for the data traffic
	packets, traffic := miscOutPacketsMeter, miscOutTrafficMeter
	packets.Mark(1)
	traffic.Mark(int64(msg.Size))

	// Send the packet to the p2p layer
	return rw.MsgReadWriter.WriteMsg(msg)
}
