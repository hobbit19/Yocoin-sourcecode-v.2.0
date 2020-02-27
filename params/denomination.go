// Authored and revised by YOC team, 2017-2018
// License placeholder #1

package params

// These are the multipliers for yoc denominations.
// Example: To get the wei value of an amount in 'douglas', use
//
//    new(big.Int).Mul(value, big.NewInt(params.Douglas))
//
const (
	Wei      = 1
	Ada      = 1e3
	Babbage  = 1e6
	Shannon  = 1e9
	Szabo    = 1e12
	Finney   = 1e15
	YOC      = 1e18
	Einstein = 1e21
	Douglas  = 1e42
)
