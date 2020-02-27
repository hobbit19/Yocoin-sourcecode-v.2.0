package nov2019

import (
	"testing"
)

func TestUTWBlocking1(t *testing.T) {
	mgr := GetNov2019MigrationManager()
	if !mgr.IfFilteringCriteria(NOV2019Black, "0xSOMERANDOMADDRESS") {
		t.Fail()
	}
	if !mgr.IfFilteringCriteria("0xSOMERANDOMADDRESS", NOV2019Black) {
		t.Fail()
	}
	if !mgr.IfFilteringCriteria("0xSOMERANDOMADDRESS", NOV2019Block) {
		t.Fail()
	}
}

/*
с Good адреса можно слать

func TestUTWBlocking2(t *testing.T) {
	mgr := GetNov2019MigrationManager()
	if !mgr.IfFilteringCriteria(NOV2019Good, "0xSOMERANDOMADDRESS") {
		t.Fail()
	}
}
*/

func TestUTWNormalTxWorks(t *testing.T) {
	mgr := GetNov2019MigrationManager()
	if mgr.IfFilteringCriteria("0xSOMERANDOMADDRESS", "0xSOMEOTHERADDRESS") {
		t.Fail()
	}
}
