// Authored and revised by YOC team, 2017-2018
// License placeholder #1

package yoc

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/common/hexutil"
	"github.com/Yocoin15/Yocoin_Sources/consensus/yochash"
	"github.com/Yocoin15/Yocoin_Sources/core"
	"github.com/Yocoin15/Yocoin_Sources/params"
	"github.com/Yocoin15/Yocoin_Sources/yoc/downloader"
	"github.com/Yocoin15/Yocoin_Sources/yoc/gasprice"
)

// DefaultConfig contains default settings for use on the YoCoin main net.
var DefaultConfig = Config{
	SyncMode: downloader.FastSync,
	Yochash: yochash.Config{
		CacheDir:       "yochash",
		CachesInMem:    2,
		CachesOnDisk:   3,
		DatasetsInMem:  1,
		DatasetsOnDisk: 2,
	},
	NetworkId:     13,
	LightPeers:    100,
	DatabaseCache: 768,
	TrieCache:     256,
	TrieTimeout:   60 * time.Minute,
	GasPrice:      big.NewInt(18 * params.Shannon),

	TxPool: core.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     20,
		Percentile: 60,
	},
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.Yochash.DatasetDir = filepath.Join(home, "AppData", "Yochash")
	} else {
		DefaultConfig.Yochash.DatasetDir = filepath.Join(home, ".yochash")
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the YoCoin main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode
	NoPruning bool

	// Light client options
	LightServ  int `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightPeers int `toml:",omitempty"` // Maximum number of LES client peers

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	TrieCache          int
	TrieTimeout        time.Duration

	// Mining-related options
	Yocbase      common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    []byte         `toml:",omitempty"`
	GasPrice     *big.Int

	// Yochash options
	Yochash yochash.Config

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
