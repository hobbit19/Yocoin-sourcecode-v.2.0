// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package yocclient

import "github.com/Yocoin15/Yocoin_Sources"

// Verify that Client implements the yocoin interfaces.
var (
	_ = yocoin.ChainReader(&Client{})
	_ = yocoin.TransactionReader(&Client{})
	_ = yocoin.ChainStateReader(&Client{})
	_ = yocoin.ChainSyncReader(&Client{})
	_ = yocoin.ContractCaller(&Client{})
	_ = yocoin.GasEstimator(&Client{})
	_ = yocoin.GasPricer(&Client{})
	_ = yocoin.LogFilterer(&Client{})
	_ = yocoin.PendingStateReader(&Client{})
	// _ = yocoin.PendingStateEventer(&Client{})
	_ = yocoin.PendingContractCaller(&Client{})
)
