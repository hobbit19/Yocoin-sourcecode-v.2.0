// Authored and revised by YOC team, 2015-2018
// License placeholder #1

package abi

import (
	"math/big"
	"reflect"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/math"
)

var (
	bigT      = reflect.TypeOf(&big.Int{})
	derefbigT = reflect.TypeOf(big.Int{})
	uint8T    = reflect.TypeOf(uint8(0))
	uint16T   = reflect.TypeOf(uint16(0))
	uint32T   = reflect.TypeOf(uint32(0))
	uint64T   = reflect.TypeOf(uint64(0))
	int8T     = reflect.TypeOf(int8(0))
	int16T    = reflect.TypeOf(int16(0))
	int32T    = reflect.TypeOf(int32(0))
	int64T    = reflect.TypeOf(int64(0))
	addressT  = reflect.TypeOf(common.Address{})
)

// U256 converts a big Int into a 256bit YVM number.
func U256(n *big.Int) []byte {
	return math.PaddedBigBytes(math.U256(n), 32)
}
