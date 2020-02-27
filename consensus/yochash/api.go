// Authored and revised by YOC team, 2018
// License placeholder #1

package yochash

import (
	"errors"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/hexutil"
	"github.com/Yocoin15/Yocoin_Sources/core/types"
)

var errYochashStopped = errors.New("yochash stopped")

// API exposes yochash related methods for the RPC interface.
type API struct {
	yochash *Yochash // Make sure the mode of yochash is normal.
}

// GetWork returns a work package for external miner.
//
// The work package consists of 3 strings:
//   result[0] - 32 bytes hex encoded current block header pow-hash
//   result[1] - 32 bytes hex encoded seed hash used for DAG
//   result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (api *API) GetWork() ([3]string, error) {
	if api.yochash.config.PowMode != ModeNormal && api.yochash.config.PowMode != ModeTest {
		return [3]string{}, errors.New("not supported")
	}

	var (
		workCh = make(chan [3]string, 1)
		errc   = make(chan error, 1)
	)

	select {
	case api.yochash.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.yochash.exitCh:
		return [3]string{}, errYochashStopped
	}

	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [3]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
func (api *API) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	if api.yochash.config.PowMode != ModeNormal && api.yochash.config.PowMode != ModeTest {
		return false
	}

	var errc = make(chan error, 1)

	select {
	case api.yochash.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      hash,
		errc:      errc,
	}:
	case <-api.yochash.exitCh:
		return false
	}

	err := <-errc
	return err == nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate.
// This enables the node to report the combined hash rate of all miners
// which submit work through this node.
//
// It accepts the miner hash rate and an identifier which must be unique
// between nodes.
func (api *API) SubmitHashRate(rate hexutil.Uint64, id common.Hash) bool {
	if api.yochash.config.PowMode != ModeNormal && api.yochash.config.PowMode != ModeTest {
		return false
	}

	var done = make(chan struct{}, 1)

	select {
	case api.yochash.submitRateCh <- &hashrate{done: done, rate: uint64(rate), id: id}:
	case <-api.yochash.exitCh:
		return false
	}

	// Block until hash rate submitted successfully.
	<-done

	return true
}

// GetHashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) GetHashrate() uint64 {
	return uint64(api.yochash.Hashrate())
}
