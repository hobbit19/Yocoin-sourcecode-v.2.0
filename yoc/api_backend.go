// Authored and revised by YOC team, 2015-2018
// License placeholder #1

package yoc

import (
	"context"
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
	"github.com/Yocoin15/Yocoin_Sources/params"
	"github.com/Yocoin15/Yocoin_Sources/rpc"
	"github.com/Yocoin15/Yocoin_Sources/yoc/downloader"
	"github.com/Yocoin15/Yocoin_Sources/yoc/gasprice"
	"github.com/Yocoin15/Yocoin_Sources/yocdb"
)

// YocAPIBackend implements yocapi.Backend for full nodes
type YocAPIBackend struct {
	yoc *YoCoin
	gpo *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *YocAPIBackend) ChainConfig() *params.ChainConfig {
	return b.yoc.chainConfig
}

func (b *YocAPIBackend) CurrentBlock() *types.Block {
	return b.yoc.blockchain.CurrentBlock()
}

func (b *YocAPIBackend) SetHead(number uint64) {
	b.yoc.protocolManager.downloader.Cancel()
	b.yoc.blockchain.SetHead(number)
}

func (b *YocAPIBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.yoc.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.yoc.blockchain.CurrentBlock().Header(), nil
	}
	return b.yoc.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *YocAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.yoc.blockchain.GetHeaderByHash(hash), nil
}

func (b *YocAPIBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.yoc.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.yoc.blockchain.CurrentBlock(), nil
	}
	return b.yoc.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *YocAPIBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.yoc.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.yoc.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *YocAPIBackend) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.yoc.blockchain.GetBlockByHash(hash), nil
}

func (b *YocAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.yoc.chainDb, hash); number != nil {
		return rawdb.ReadReceipts(b.yoc.chainDb, hash, *number), nil
	}
	return nil, nil
}

func (b *YocAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	number := rawdb.ReadHeaderNumber(b.yoc.chainDb, hash)
	if number == nil {
		return nil, nil
	}
	receipts := rawdb.ReadReceipts(b.yoc.chainDb, hash, *number)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *YocAPIBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.yoc.blockchain.GetTdByHash(blockHash)
}

func (b *YocAPIBackend) GetYVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.YVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256, header.Number)
	vmError := func() error { return nil }

	context := core.NewYVMContext(msg, header, b.yoc.BlockChain(), nil)
	return vm.NewYVM(context, state, b.yoc.chainConfig, vmCfg), vmError, nil
}

func (b *YocAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.yoc.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *YocAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.yoc.BlockChain().SubscribeChainEvent(ch)
}

func (b *YocAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.yoc.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *YocAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.yoc.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *YocAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.yoc.BlockChain().SubscribeLogsEvent(ch)
}

func (b *YocAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.yoc.txPool.AddLocal(signedTx)
}

func (b *YocAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.yoc.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *YocAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.yoc.txPool.Get(hash)
}

func (b *YocAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.yoc.txPool.State().GetNonce(addr), nil
}

func (b *YocAPIBackend) Stats() (pending int, queued int) {
	return b.yoc.txPool.Stats()
}

func (b *YocAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.yoc.TxPool().Content()
}

func (b *YocAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.yoc.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *YocAPIBackend) Downloader() *downloader.Downloader {
	return b.yoc.Downloader()
}

func (b *YocAPIBackend) ProtocolVersion() int {
	return b.yoc.YocVersion()
}

func (b *YocAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *YocAPIBackend) ChainDb() yocdb.Database {
	return b.yoc.ChainDb()
}

func (b *YocAPIBackend) EventMux() *event.TypeMux {
	return b.yoc.EventMux()
}

func (b *YocAPIBackend) AccountManager() *accounts.Manager {
	return b.yoc.AccountManager()
}

func (b *YocAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.yoc.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *YocAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.yoc.bloomRequests)
	}
}

func (b *YocAPIBackend) Miner() *miner.Miner {
	return b.yoc.miner
}
