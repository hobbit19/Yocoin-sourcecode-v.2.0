// Authored and revised by YOC team, 2017-2018
// License placeholder #1

package yochash

import (
	crand "crypto/rand"
	"errors"
	"github.com/Yocoin15/Yocoin_Sources/nov2019"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/consensus"
	"github.com/Yocoin15/Yocoin_Sources/core/types"
	"github.com/Yocoin15/Yocoin_Sources/log"
)

var (
	errNoMiningWork      = errors.New("no mining work available yet")
	errInvalidSealResult = errors.New("invalid or stale proof-of-work solution")
)

// Seal implements consensus.Engine, attempting to find a nonce that satisfies
// the block's difficulty requirements.
func (yochash *Yochash) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// If we're running a fake PoW, simply return a 0 nonce immediately
	if yochash.config.PowMode == ModeFake || yochash.config.PowMode == ModeFullFake {
		header := block.Header()
		header.Nonce, header.MixDigest = types.BlockNonce{}, common.Hash{}
		log.Info(nov2019.LOG_PREFIX+"Seal branch (1)", "nov2019", true)
		return block.WithSeal(header), nil
	}
	// If we're running a shared PoW, delegate sealing to it
	if yochash.shared != nil {
		return yochash.shared.Seal(chain, block, stop)
	}
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})

	yochash.lock.Lock()
	threads := yochash.threads
	if yochash.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			yochash.lock.Unlock()
			log.Info(nov2019.LOG_PREFIX+"Seal branch (2)", "nov2019", true)
			return nil, err
		}
		yochash.rand = rand.New(rand.NewSource(seed.Int64()))
	}
	yochash.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	// Push new work to remote sealer
	if yochash.workCh != nil {
		yochash.workCh <- block
	}
	var pend sync.WaitGroup
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			yochash.mine(block, id, nonce, abort, yochash.resultCh)
		}(i, uint64(yochash.rand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	var result *types.Block
	select {
	case <-stop:
		// Outside abort, stop all miner threads
		close(abort)
	case result = <-yochash.resultCh:
		// One of the threads found a block, abort all others
		close(abort)
	case <-yochash.update:
		// Thread count was changed on user request, restart
		close(abort)
		pend.Wait()

		a, b := yochash.Seal(chain, block, stop)
		log.Info(nov2019.LOG_PREFIX+"Seal branch (3)", "nov2019", true, "block", a, "err", b)
		return a, b
	}
	// Wait for all miners to terminate and return the block
	pend.Wait()
	log.Info(nov2019.LOG_PREFIX+"Seal branch (4)", "nov2019", true)
	return result, nil
}

// mine is the actual proof-of-work miner that searches for a nonce starting from
// seed that results in correct final block difficulty.
func (yochash *Yochash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	// Extract some data from the header
	var (
		header  = block.Header()
		hash    = header.HashNoNonce().Bytes()
		target  = new(big.Int).Div(maxUint256, header.Difficulty)
		number  = header.Number.Uint64()
		dataset = yochash.dataset(number)
	)
	// Start generating random nonces until we abort or find a good one
	var (
		attempts = int64(0)
		nonce    = seed
	)
	logger := log.New("miner", id)
	logger.Trace("Started yochash search for new nonces", "seed", seed)
search:
	for {
		select {
		case <-abort:
			// Mining terminated, update stats and abort
			logger.Trace("Yochash nonce search aborted", "attempts", nonce-seed)
			yochash.hashrate.Mark(attempts)
			break search

		default:
			// We don't have to update hash rate on every nonce, so update after after 2^X nonces
			attempts++
			if (attempts % (1 << 15)) == 0 {
				yochash.hashrate.Mark(attempts)
				attempts = 0
			}
			// Compute the PoW value of this nonce
			digest, result := hashimotoFull(dataset.dataset, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				// Correct nonce found, create a new header with it
				header = types.CopyHeader(header)
				header.Nonce = types.EncodeNonce(nonce)
				header.MixDigest = common.BytesToHash(digest)

				// Seal and return a block (if still needed)
				select {
				case found <- block.WithSeal(header):
					logger.Trace("Yochash nonce found and reported", "attempts", nonce-seed, "nonce", nonce)
				case <-abort:
					logger.Trace("Yochash nonce found but discarded", "attempts", nonce-seed, "nonce", nonce)
				}
				break search
			}
			nonce++
		}
	}
	// Datasets are unmapped in a finalizer. Ensure that the dataset stays live
	// during sealing so it's not unmapped while being read.
	runtime.KeepAlive(dataset)
}

// remote starts a standalone goroutine to handle remote mining related stuff.
func (yochash *Yochash) remote() {
	var (
		works       = make(map[common.Hash]*types.Block)
		rates       = make(map[common.Hash]hashrate)
		currentWork *types.Block
	)

	// getWork returns a work package for external miner.
	//
	// The work package consists of 3 strings:
	//   result[0], 32 bytes hex encoded current block header pow-hash
	//   result[1], 32 bytes hex encoded seed hash used for DAG
	//   result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
	getWork := func() ([3]string, error) {
		var res [3]string
		if currentWork == nil {
			return res, errNoMiningWork
		}
		res[0] = currentWork.HashNoNonce().Hex()
		res[1] = common.BytesToHash(SeedHash(currentWork.NumberU64())).Hex()

		// Calculate the "target" to be returned to the external sealer.
		n := big.NewInt(1)
		n.Lsh(n, 255)
		n.Div(n, currentWork.Difficulty())
		n.Lsh(n, 1)
		res[2] = common.BytesToHash(n.Bytes()).Hex()

		// Trace the seal work fetched by remote sealer.
		works[currentWork.HashNoNonce()] = currentWork
		return res, nil
	}

	// submitWork verifies the submitted pow solution, returning
	// whether the solution was accepted or not (not can be both a bad pow as well as
	// any other error, like no pending work or stale mining result).
	submitWork := func(nonce types.BlockNonce, mixDigest common.Hash, hash common.Hash) bool {
		// Make sure the work submitted is present
		block := works[hash]
		if block == nil {
			log.Info("Work submitted but none pending", "hash", hash)
			return false
		}

		// Verify the correctness of submitted result.
		header := block.Header()
		header.Nonce = nonce
		header.MixDigest = mixDigest
		if err := yochash.VerifySeal(nil, header); err != nil {
			log.Warn("Invalid proof-of-work submitted", "hash", hash, "err", err)
			return false
		}

		// Make sure the result channel is created.
		if yochash.resultCh == nil {
			log.Warn("Yochash result channel is empty, submitted mining result is rejected")
			return false
		}

		// Solutions seems to be valid, return to the miner and notify acceptance.
		select {
		case yochash.resultCh <- block.WithSeal(header):
			delete(works, hash)
			return true
		default:
			log.Info("Work submitted is stale", "hash", hash)
			return false
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case block := <-yochash.workCh:
			if currentWork != nil && block.ParentHash() != currentWork.ParentHash() {
				// Start new round mining, throw out all previous work.
				works = make(map[common.Hash]*types.Block)
			}
			// Update current work with new received block.
			// Note same work can be past twice, happens when changing CPU threads.
			currentWork = block

		case work := <-yochash.fetchWorkCh:
			// Return current mining work to remote miner.
			miningWork, err := getWork()
			if err != nil {
				work.errc <- err
			} else {
				work.res <- miningWork
			}

		case result := <-yochash.submitWorkCh:
			// Verify submitted PoW solution based on maintained mining blocks.
			if submitWork(result.nonce, result.mixDigest, result.hash) {
				result.errc <- nil
			} else {
				result.errc <- errInvalidSealResult
			}

		case result := <-yochash.submitRateCh:
			// Trace remote sealer's hash rate by submitted value.
			rates[result.id] = hashrate{rate: result.rate, ping: time.Now()}
			close(result.done)

		case req := <-yochash.fetchRateCh:
			// Gather all hash rate submitted by remote sealer.
			var total uint64
			for _, rate := range rates {
				// this could overflow
				total += rate.rate
			}
			req <- total

		case <-ticker.C:
			// Clear stale submitted hash rate.
			for id, rate := range rates {
				if time.Since(rate.ping) > 10*time.Second {
					delete(rates, id)
				}
			}

		case errc := <-yochash.exitCh:
			// Exit remote loop if yochash is closed and return relevant error.
			errc <- nil
			log.Trace("Yochash remote sealer is exiting")
			return
		}
	}
}
