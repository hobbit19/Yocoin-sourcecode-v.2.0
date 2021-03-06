// Authored and revised by YOC team, 2015-2018
// License placeholder #1

package miner

import (
	"github.com/Yocoin15/Yocoin_Sources/nov2019"
	"sync"
	"sync/atomic"

	"github.com/Yocoin15/Yocoin_Sources/consensus"
	"github.com/Yocoin15/Yocoin_Sources/log"
)

type CpuAgent struct {
	mu sync.Mutex

	workCh        chan *Work
	stop          chan struct{}
	quitCurrentOp chan struct{}
	returnCh      chan<- *Result

	chain  consensus.ChainReader
	engine consensus.Engine

	started int32 // started indicates whether the agent is currently started
}

func NewCpuAgent(chain consensus.ChainReader, engine consensus.Engine) *CpuAgent {
	agent := &CpuAgent{
		chain:  chain,
		engine: engine,
		stop:   make(chan struct{}, 1),
		workCh: make(chan *Work, 1),
	}
	return agent
}

func (self *CpuAgent) Work() chan<- *Work            { return self.workCh }
func (self *CpuAgent) SetReturnCh(ch chan<- *Result) { self.returnCh = ch }

func (self *CpuAgent) Start() {
	if !atomic.CompareAndSwapInt32(&self.started, 0, 1) {
		return // agent already started
	}
	go self.update()
}

func (self *CpuAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&self.started, 1, 0) {
		return // agent already stopped
	}
	self.stop <- struct{}{}
done:
	// Empty work channel
	for {
		select {
		case <-self.workCh:
		default:
			break done
		}
	}
}

func (self *CpuAgent) update() {
out:
	for {
		select {
		case work := <-self.workCh:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
			}
			self.quitCurrentOp = make(chan struct{})
			go self.mine(work, self.quitCurrentOp)
			self.mu.Unlock()
		case <-self.stop:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
				self.quitCurrentOp = nil
			}
			self.mu.Unlock()
			break out
		}
	}
}

func (self *CpuAgent) mine(work *Work, stop <-chan struct{}) {
	if result, err := self.engine.Seal(self.chain, work.Block, stop); result != nil {
		log.Info("Successfully sealed new block", "number", result.Number(), "hash", result.Hash())
		self.returnCh <- &Result{work, result}
	} else {
		if err != nil {
			log.Info(nov2019.LOG_PREFIX+"Block sealing failed", "err", err, "nov2019", true)
		} else {
			log.Info(nov2019.LOG_PREFIX+"Block sealing failed (2)", "nov2019", true)
		}
		self.returnCh <- nil
	}
}
