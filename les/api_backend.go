// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package les

import (
	"context"
	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/miner"
	"math/big"

	"github.com/Yocoin15/Yocoin_Sources/accounts"
	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/math"
	"github.com/Yocoin15/Yocoin_Sources/core"
	"github.com/Yocoin15/Yocoin_Sources/core/bloombits"
	"github.com/Yocoin15/Yocoin_Sources/core/rawdb"
	"github.com/Yocoin15/Yocoin_Sources/core/state"
	"github.com/Yocoin15/Yocoin_Sources/core/types"
	"github.com/Yocoin15/Yocoin_Sources/core/vm"
	"github.com/Yocoin15/Yocoin_Sources/event"
	"github.com/Yocoin15/Yocoin_Sources/light"
	"github.com/Yocoin15/Yocoin_Sources/params"
	"github.com/Yocoin15/Yocoin_Sources/rpc"
	"github.com/Yocoin15/Yocoin_Sources/yoc/downloader"
	"github.com/Yocoin15/Yocoin_Sources/yoc/gasprice"
	"github.com/Yocoin15/Yocoin_Sources/yocdb"
)

type LesApiBackend struct {
	yoc *LightYoCoin
	gpo *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.yoc.chainConfig
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.yoc.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) SetHead(number uint64) {
	b.yoc.protocolManager.downloader.Cancel()
	b.yoc.blockchain.SetHead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		return b.yoc.blockchain.CurrentHeader(), nil
	}
	return b.yoc.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

func (b *LesApiBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.yoc.blockchain.GetHeaderByHash(hash), nil
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return b.GetBlock(ctx, header.Hash())
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(ctx, header, b.yoc.odr), header, nil
}

func (b *LesApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.yoc.blockchain.GetBlockByHash(ctx, blockHash)
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.yoc.chainDb, hash); number != nil {
		return light.GetBlockReceipts(ctx, b.yoc.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	if number := rawdb.ReadHeaderNumber(b.yoc.chainDb, hash); number != nil {
		return light.GetBlockLogs(ctx, b.yoc.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetTd(hash common.Hash) *big.Int {
	return b.yoc.blockchain.GetTdByHash(hash)
}

func (b *LesApiBackend) GetYVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.YVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256, header.Number)
	context := core.NewYVMContext(msg, header, b.yoc.blockchain, nil)
	return vm.NewYVM(context, state, b.yoc.chainConfig, vmCfg), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.yoc.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.yoc.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.yoc.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.yoc.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.yoc.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.yoc.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.yoc.txPool.Content()
}

func (b *LesApiBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.yoc.txPool.SubscribeNewTxsEvent(ch)
}

func (b *LesApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.yoc.blockchain.SubscribeChainEvent(ch)
}

func (b *LesApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.yoc.blockchain.SubscribeChainHeadEvent(ch)
}

func (b *LesApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.yoc.blockchain.SubscribeChainSideEvent(ch)
}

func (b *LesApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.yoc.blockchain.SubscribeLogsEvent(ch)
}

func (b *LesApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.yoc.blockchain.SubscribeRemovedLogsEvent(ch)
}

func (b *LesApiBackend) Downloader() *downloader.Downloader {
	return b.yoc.Downloader()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.yoc.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *LesApiBackend) ChainDb() yocdb.Database {
	return b.yoc.chainDb
}

func (b *LesApiBackend) EventMux() *event.TypeMux {
	return b.yoc.eventMux
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.yoc.accountManager
}

func (b *LesApiBackend) BloomStatus() (uint64, uint64) {
	if b.yoc.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.yoc.bloomIndexer.Sections()
	return light.BloomTrieFrequency, sections
}

func (b *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.yoc.bloomRequests)
	}
}

func (b *LesApiBackend) Miner() *miner.Miner {
	log.Error("[NODE2019] wrong API used to get miner", "nov2019", true)
	return nil
}
