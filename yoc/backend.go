// Authored and revised by YOC team, 2014-2018
// License placeholder #1

// Package yoc implements the YoCoin protocol.
package yoc

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/Yocoin15/Yocoin_Sources/accounts"
	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/hexutil"
	"github.com/Yocoin15/Yocoin_Sources/consensus"
	"github.com/Yocoin15/Yocoin_Sources/consensus/clique"
	"github.com/Yocoin15/Yocoin_Sources/consensus/yochash"
	"github.com/Yocoin15/Yocoin_Sources/core"
	"github.com/Yocoin15/Yocoin_Sources/core/bloombits"
	"github.com/Yocoin15/Yocoin_Sources/core/rawdb"
	"github.com/Yocoin15/Yocoin_Sources/core/types"
	"github.com/Yocoin15/Yocoin_Sources/core/vm"
	"github.com/Yocoin15/Yocoin_Sources/event"
	"github.com/Yocoin15/Yocoin_Sources/internal/yocapi"
	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/miner"
	"github.com/Yocoin15/Yocoin_Sources/node"
	"github.com/Yocoin15/Yocoin_Sources/p2p"
	"github.com/Yocoin15/Yocoin_Sources/params"
	"github.com/Yocoin15/Yocoin_Sources/rlp"
	"github.com/Yocoin15/Yocoin_Sources/rpc"
	"github.com/Yocoin15/Yocoin_Sources/yoc/downloader"
	"github.com/Yocoin15/Yocoin_Sources/yoc/filters"
	"github.com/Yocoin15/Yocoin_Sources/yoc/gasprice"
	"github.com/Yocoin15/Yocoin_Sources/yocdb"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// YoCoin implements the YoCoin full node service.
type YoCoin struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the YoCoin

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb yocdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *YocAPIBackend

	miner    *miner.Miner
	gasPrice *big.Int
	yocbase  common.Address

	networkID     uint64
	netRPCService *yocapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and yocbase)
}

func (s *YoCoin) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new YoCoin object (including the
// initialisation of the common YoCoin object)
func New(ctx *node.ServiceContext, config *Config) (*YoCoin, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run Yocoin in light sync mode, use les")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	yoc := &YoCoin{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Yochash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkID:      config.NetworkId,
		gasPrice:       config.GasPrice,
		yocbase:        config.Yocbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising YoCoin protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := rawdb.ReadDatabaseVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	yoc.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, yoc.chainConfig, yoc.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		yoc.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	yoc.bloomIndexer.Start(yoc.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	yoc.txPool = core.NewTxPool(config.TxPool, yoc.chainConfig, yoc.blockchain)

	if yoc.protocolManager, err = NewProtocolManager(yoc.chainConfig, config.SyncMode, config.NetworkId, yoc.eventMux, yoc.txPool, yoc.engine, yoc.blockchain, chainDb); err != nil {
		return nil, err
	}

	yoc.miner = miner.New(yoc, yoc.chainConfig, yoc.EventMux(), yoc.engine)
	yoc.miner.SetExtra(makeExtraData(config.ExtraData))

	yoc.APIBackend = &YocAPIBackend{yoc, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	yoc.APIBackend.gpo = gasprice.NewOracle(yoc.APIBackend, gpoParams)

	return yoc, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"yocoin",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (yocdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*yocdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an YoCoin service
func CreateConsensusEngine(ctx *node.ServiceContext, config *yochash.Config, chainConfig *params.ChainConfig, db yocdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case yochash.ModeFake:
		log.Warn("Yochash used in fake mode")
		return yochash.NewFaker()
	case yochash.ModeTest:
		log.Warn("Yochash used in test mode")
		return yochash.NewTester()
	case yochash.ModeShared:
		log.Warn("Yochash used in shared mode")
		return yochash.NewShared()
	default:
		engine := yochash.New(yochash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs return the collection of RPC services the yocoin package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *YoCoin) APIs() []rpc.API {
	apis := yocapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "yoc",
			Version:   "1.0",
			Service:   NewPublicYoCoinAPI(s),
			Public:    true,
		}, {
			Namespace: "yoc",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "yoc",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "yoc",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.APIBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *YoCoin) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *YoCoin) Yocbase() (eb common.Address, err error) {
	s.lock.RLock()
	yocbase := s.yocbase
	s.lock.RUnlock()

	if yocbase != (common.Address{}) {
		return yocbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			yocbase := accounts[0].Address

			s.lock.Lock()
			s.yocbase = yocbase
			s.lock.Unlock()

			log.Info("Yocbase automatically configured", "address", yocbase)
			return yocbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("yocbase must be explicitly specified")
}

// SetYocbase sets the mining reward address.
func (s *YoCoin) SetYocbase(yocbase common.Address) {
	s.lock.Lock()
	s.yocbase = yocbase
	s.lock.Unlock()

	s.miner.SetYocbase(yocbase)
}

func (s *YoCoin) StartMining(local bool) error {
	eb, err := s.Yocbase()
	if err != nil {
		log.Error("Cannot start mining without yocbase", "err", err)
		return fmt.Errorf("yocbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Yocbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so none will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *YoCoin) StopMining()         { s.miner.Stop() }
func (s *YoCoin) IsMining() bool      { return s.miner.Mining() }
func (s *YoCoin) Miner() *miner.Miner { return s.miner }

func (s *YoCoin) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *YoCoin) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *YoCoin) TxPool() *core.TxPool               { return s.txPool }
func (s *YoCoin) EventMux() *event.TypeMux           { return s.eventMux }
func (s *YoCoin) Engine() consensus.Engine           { return s.engine }
func (s *YoCoin) ChainDb() yocdb.Database            { return s.chainDb }
func (s *YoCoin) IsListening() bool                  { return true } // Always listening
func (s *YoCoin) YocVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *YoCoin) NetVersion() uint64                 { return s.networkID }
func (s *YoCoin) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *YoCoin) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// YoCoin protocol implementation.
func (s *YoCoin) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = yocapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// YoCoin protocol.
func (s *YoCoin) Stop() error {
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.engine.Close()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)
	return nil
}
