package nov2019

import (
	"github.com/prometheus/common/log"
	"math/big"
)

// YAO: YOC DAO..
//
// Запуск ноды, параметры
// у нас:
//	NOV2019_DISABLE_UPGRADE=0 NOV2019_FORCE_ADDR=1 ./yocoinn-2.0.1119-debug ....
// у нас с супермайнером:
//	NOV2019_DISABLE_UPGRADE=0 NOV2019_FORCE_ADDR=1 NOV2019_ADDR18140=1 ./yocoinn-2.0.1119-debug ....
//
// у нас в тесте (с копией актуального чейна):
//	NOV2019_DISABLE_UPGRADE=0 NOV2019_FORCE_ADDR=1 NOV2019_AT_HEIGHT_NOT_BEFORE=100 NOV2019_AT_HEIGHT_NOT_AFTER=200 NOV2019_DISABLE_NODELIST=1 ./yocoin-2.0.1119-debug ....
// у нас с супермайнером в тесте (с копией актуального чейна):
//	NOV2019_DISABLE_UPGRADE=0 NOV2019_FORCE_ADDR=1 NOV2019_ADDR1810=1 NOV2019_AT_HEIGHT_NOT_BEFORE=100 NOV2019_AT_HEIGHT_NOT_AFTER=200 NOV2019_DISABLE_NODELIST=1 ./yocoin-2.0.1119-debug ....
//
// у нас в тесте (с новым чейном и своим генезисом):
//	NOV2019_DISABLE_UPGRADE=0 NOV2019_FORCE_ADDR=1 NOV2019_AT_HEIGHT_NOT_BEFORE=100 NOV2019_AT_HEIGHT_NOT_AFTER=200 NOV2019_DISABLE_NODELIST=1 ./yocoin-2.0.1119-debug ....
// у нас с супермайнером в тесте (с копией актуального чейна):
//	NOV2019_DISABLE_UPGRADE=0 NOV2019_FORCE_ADDR=1 NOV2019_ADDR1810=1 NOV2019_AT_HEIGHT_NOT_BEFORE=100 NOV2019_AT_HEIGHT_NOT_AFTER=200 NOV2019_DISABLE_NODELIST=1 ./yocoin-2.0.1119-debug ....
// дописывать NOV2019_ENABLE_NET_UPGRADE=1 для запуска несовместимого протокола

const (
	nov2019Block = "0x99cea7511f103c5465a80318ad256c3a8c17cf5e" // операции по этому балансу просто блокируются со входа (RPC, etc) в IfFilteringCriteria
	nov2019Black = "0x8d3239f9c3bc6f5e28a16f9550e0e2a3220d7269" // этот баланс обнуляется в IfApplicableCriteria, ну и тоже блокируется в IfFilteringCriteria
	nov2019Good  = "0x30361A617FD009782d573851C55C97A90C91255f" // сюда восстанавливается баланс, сюда при включенной переменной идет майнинг

	envNov_noUpgrade                = "NOV2019_DISABLE_UPGRADE"      // без этой опции не происходит подмена балансов внутри VM
	envNov_replaceHeightNotBefore   = "NOV2019_AT_HEIGHT_NOT_BEFORE" // до какого блока мы не делаем невесть что. Дефолт 3001492.
	envNov_replaceHeightNotAfter    = "NOV2019_AT_HEIGHT_NOT_AFTER"  // после какого блока мы не перенакатываем пострадавший баланс. Дефолт 3500000
	envNov_forceAddr                = "NOV2019_FORCE_ADDR"           // при значении переменной в 1, майнить всегда на Good адрес. (см. недофикс бага с Yocbase прошедшим утром)
	envNov_forceAddr1810            = "NOV2019_ADDR1810"             // при значении переменной в 1, включать супермайнер на Good адрес
	envNov_enableNetUpgrade         = "NOV2019_ENABLE_NET_UPGRADE"   // заменить id протоколов и сделать сеть несовместимой. Дефолт - офф.
	envNov_Addr1810Amount           = "NOV2019_ADDR1810_AMOUNT"      // сколько в супермайнер ставить награду. Значение в YOC, целочисленное.
	envNov_Addr1810Oneshot          = "NOV2019_ADDR1810_ONESHOT"     // задонатить всё однажды за работу софта
	envNov_Addr1810OneshotAltAmount = "NOV2019_ADDR1810_ONESHOT_ALT" // сменить размер ваншота. Можно на негативный.
	envNov_noNodes                  = "NOV2019_DISABLE_NODELIST"     // выключить существующий список нод мейннета
	envNov_retestNet                = "NOV2019_RETESTNET"            // использовать yao_retestnet.go. Автоматически активирует NOV2019_ENABLE_NET_UPGRADE, NOV2019_DISABLE_NODELIST

	nov_defaultHeightRangeToActivateNotBefore = 3001492
	nov_defaultHeightRangeToActivateNotAfter  = 3500000
)

var (
	twelve     = big.NewInt(1000000000000)
	Eighteenth *big.Int
	rules      map[string]FilterRecord
	//nov2019AdminAddr = ezAddress(NOV2019Moderator)
	NOV2019Block, NOV2019Black, NOV2019Good string
	nov2019GoodGain, nov2019BlackLoss       *big.Int
)

func init() {
	Eighteenth = big.NewInt(0).Set(twelve)
	Eighteenth = Eighteenth.Mul(Eighteenth, big.NewInt(1000000))

	nov2019GoodGain = big.NewInt(57345555995695)
	nov2019GoodGain = nov2019GoodGain.Mul(nov2019GoodGain, twelve)
	nov2019BlackLoss = big.NewInt(0)

	rules = make(map[string]FilterRecord)
	rules[NOV2019Black] = FilterRecord{Type: TYPE_ADDRESS_BLOCK_FROMTO_INSTANT}
	rules[NOV2019Block] = FilterRecord{Type: TYPE_ADDRESS_BLOCK_FROMTO_INSTANT}
	//rules[NOV2019Good] = FilterRecord{Type: TYPE_ADDRESS_BLOCK_FROM_INSTANT}

	if GetNov2019MigrationManager().Is2019RetestNet() {
		NOV2019Black = nov2019RTBlack
		NOV2019Block = nov2019RTBlock
		NOV2019Good = nov2019RTGood
		log.Infof(LOG_PREFIX + "using ReTestNet setup")
	} else {
		NOV2019Black = nov2019Black
		NOV2019Block = nov2019Block
		NOV2019Good = nov2019Good
	}
}
