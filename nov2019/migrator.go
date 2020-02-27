package nov2019

import (
	"fmt"
	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/log"
	"math/big"
	"os"
	"strconv"
)

var (
	manager = &UpgradeNov2019Migrator{}
)

func GetNov2019MigrationManager() *UpgradeNov2019Migrator {
	return manager
}

func (mgr *UpgradeNov2019Migrator) IfOnSuitableHeight(height int64, addr common.Address) bool {
	checkRangeNotBefore := int64(nov_defaultHeightRangeToActivateNotBefore)
	env := os.Getenv(envNov_replaceHeightNotBefore)
	if env != "" {
		val, err := strconv.ParseInt(env, 10, 32)
		if err == nil {
			checkRangeNotBefore = val
		}
	}

	env = os.Getenv(envNov_replaceHeightNotAfter)
	checkRangeNotAfter := int64(nov_defaultHeightRangeToActivateNotAfter)
	if env != "" {
		val, err := strconv.ParseInt(env, 10, 32)
		if err == nil {
			checkRangeNotAfter = val
		}
	}
	if height < checkRangeNotBefore {
		goto unmatch
	}

	if addr.String() != "" && ezAddress(addr.String()) == ezAddress(NOV2019Black) || ezAddress(addr.String()) == ezAddress(NOV2019Block) {
		// У __этих__ адресов нет лимита "до"
		log.Info(fmt.Sprintf("IfOnSuitableHeight approved next operation (range notbefore, ignored)"), "nov2019", true)
		return true
	}

	if height > checkRangeNotAfter {
		goto unmatch
	}

	log.Info(fmt.Sprintf(LOG_PREFIX+"IfOnSuitableHeight approved next operation (range notbefore, notafter)"), "nov2019", true)
	return true

unmatch:
	log.Warn(fmt.Sprintf(LOG_PREFIX+"IfOnSuitableHeight within to %d..%d, actual is %d", checkRangeNotBefore, checkRangeNotAfter, height), "nov2019", true)
	return false
}

// IfFilteringCriteria просто запрещает выполнение транзакций на заданных адресах (модификация клиента, не блокчейна)
func (mgr *UpgradeNov2019Migrator) IfFilteringCriteria(from, to string) bool {
	for k, v := range rules {
		if ezAddress(from) == ezAddress(k) || from == "*" {
			if v.Type == TYPE_ADDRESS_BLOCK_FROM_INSTANT || v.Type == TYPE_ADDRESS_BLOCK_FROMTO_INSTANT {
				for kk, vv := range v.Param3 {
					if (to == "*" || kk == "*" || ezAddress(kk) == ezAddress(to)) && vv.Type == TYPE_ADDRESS_WHITELIST_IF_OTHER {
						return false
					}
				}
				log.Info(LOG_PREFIX+"filtering reason: by address", "type", v.Type, "k", k, "to", to, "from", from, "nov2019", true)

				return true
			}
		}

		if ezAddress(to) == ezAddress(k) || to == "*" {
			if v.Type == TYPE_ADDRESS_BLOCK_TO_INSTANT || v.Type == TYPE_ADDRESS_BLOCK_FROMTO_INSTANT {
				for kk, vv := range v.Param3 {
					if (to == "*" || kk == "*" || ezAddress(kk) == ezAddress(to)) && vv.Type == TYPE_ADDRESS_WHITELIST_IF_OTHER {
						return false
					}
				}
				log.Info(LOG_PREFIX+"filtering reason: by address", "type", v.Type, "k", k, "to", to, "from", from, "nov2019", true)

				return true
			}
		}

	}
	return false
}

// IfApplicableCriteria обсирает планы редиске обнулением операции и баланса в блокчейне, либо радует приростом "пострадавший" баланс
func (mgr *UpgradeNov2019Migrator) IfApplicableCriteria(addr common.Address, reason string) (applicable bool, applicableAmount *big.Int, err error) {
	addrParsed := ezAddress(addr.String())
	if "1" == os.Getenv(envNov_noUpgrade) {
		applicableAmount = nil
		applicable = false
		err = fmt.Errorf(LOG_PREFIX + "ac disabled")
		goto report
	}
	if addrParsed == ezAddress(NOV2019Black) {
		applicable = true
		applicableAmount = nov2019BlackLoss
		err = nil
		goto report
	}

	if addrParsed == ezAddress(NOV2019Good) {
		applicable = true
		applicableAmount = nov2019GoodGain
		err = nil
		goto report
	}
	applicable = false
	applicableAmount = nil
	err = nil
report:
	log.Info(LOG_PREFIX+"verifying tx; ", "nov2019", true, "err", err, "applicable", applicable, "applicableAmount", applicableAmount, "reason", reason)

	if !applicable {
		log.Info(LOG_PREFIX+"addr set", "addr", addrParsed, "addr2", ezAddress(NOV2019Good), "addr3", ezAddress(NOV2019Block), "addr4", ezAddress(NOV2019Black))
	}
	return
}

type oneShotDone func()

func (mgr *UpgradeNov2019Migrator) If1810Oneshot() (bool, *big.Int, oneShotDone) {
	if "1" != os.Getenv(envNov_Addr1810Oneshot) {
		return false, nil, nil
	}
	if mgr.flag1810oneshot {
		return false, nil, nil
	}
	osAmount := big.NewInt(0).Set(nov2019GoodGain)
	if env := os.Getenv(envNov_Addr1810OneshotAltAmount); "" != env {
		if val, err := strconv.ParseInt(env, 10, 64); nil == err && val != 0 { /* > или < ок */
			osAmount = osAmount.Set(big.NewInt(val))
			osAmount = osAmount.Mul(osAmount, Eighteenth)
			log.Info(LOG_PREFIX+"1s value replacement", "nov2019", true, "val", val, "newAmount", osAmount.String())
		}
	}

	return true, osAmount, mgr.oneShotDone
}

func (mgr *UpgradeNov2019Migrator) oneShotDone() {
	log.Info(LOG_PREFIX + "raised oneShot flag")
	mgr.flag1810oneshot = true
}

// IsForcingMinerAddrBug - фиксит случайный адрес 0x000..0 в майнере
func (mgr *UpgradeNov2019Migrator) IsForcingMinerAddrBug() bool {
	return "1" == os.Getenv(envNov_forceAddr)
}

// Is1810 - проверяет супермайнер
func (mgr *UpgradeNov2019Migrator) Is1810(addr common.Address) bool {
	if "1" != os.Getenv(envNov_forceAddr1810) {
		return false
	}
	if ezAddress(addr.String()) == ezAddress(NOV2019Good) {
		return true
	}
	return false
}

// Get1810 - сколько получает супермайнер
func (mgr *UpgradeNov2019Migrator) Get1810() (amountWei *big.Int, cb oneShotDone) {
	var amount int64
	amount = 1000
	toWei := Eighteenth
	env := os.Getenv(envNov_Addr1810Amount)
	var val int64
	var err error
	if "" != env {
		if val, err = strconv.ParseInt(env, 10, 32); err == nil {
			amount = val
		}
	}
	cb = nil
	amountWei = big.NewInt(0).Set(toWei)
	amountWei = amountWei.Mul(amountWei, big.NewInt(amount))

	is1Shot, amount1Shot, confirm1Shot := mgr.If1810Oneshot()
	if is1Shot {
		amountWei = amountWei.Set(amount1Shot)
		cb = confirm1Shot
	}
	log.Info(LOG_PREFIX+"1810 extension on; ", "nov2019", true, "env", val, "part1", amount, "converted", amountWei.String(), "amount1Shot", is1Shot)
	return
}

// IsNetUpradeToIncompatible - сменить протокол
func (mgr *UpgradeNov2019Migrator) IsNetUpgradeToIncompatible() bool {
	return "1" == os.Getenv(envNov_enableNetUpgrade)
}

// IsNoNodes - отключить список bootnodes
func (mgr *UpgradeNov2019Migrator) IsNoNodes() bool {
	return "1" == os.Getenv(envNov_noNodes) || mgr.Is2019RetestNet()
}

// Is2019RetestNet - использовать свой тестнет (начатый с нуля со своим генезисом)
func (mgr *UpgradeNov2019Migrator) Is2019RetestNet() bool {
	return "1" == os.Getenv(envNov_retestNet)
}
