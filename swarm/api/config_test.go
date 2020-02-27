// Authored and revised by YOC team, 2016-2018
// License placeholder #1

package api

import (
	"reflect"
	"testing"

	"github.com/Yocoin15/Yocoin_Sources/common"
	"github.com/Yocoin15/Yocoin_Sources/crypto"
)

func TestConfig(t *testing.T) {

	var hexprvkey = "65138b2aa745041b372153550584587da326ab440576b2a1191dd95cee30039c"

	prvkey, err := crypto.HexToECDSA(hexprvkey)
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}

	one := NewConfig()
	two := NewConfig()

	one.LocalStoreParams = two.LocalStoreParams
	if equal := reflect.DeepEqual(one, two); !equal {
		t.Fatal("Two default configs are not equal")
	}

	one.Init(prvkey)

	//the init function should set the following fields
	if one.BzzKey == "" {
		t.Fatal("Expected BzzKey to be set")
	}
	if one.PublicKey == "" {
		t.Fatal("Expected PublicKey to be set")
	}
	if one.Swap.PayProfile.Beneficiary == (common.Address{}) && one.SwapEnabled {
		t.Fatal("Failed to correctly initialize SwapParams")
	}
	if one.ChunkDbPath == one.Path {
		t.Fatal("Failed to correctly initialize StoreParams")
	}
}
