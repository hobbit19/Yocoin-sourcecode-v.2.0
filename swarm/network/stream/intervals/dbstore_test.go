// Authored and revised by YOC team, 2018
// License placeholder #1

package intervals

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Yocoin15/Yocoin_Sources/swarm/state"
)

// TestDBStore tests basic functionality of DBStore.
func TestDBStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "intervals_test_db_store")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	store, err := state.NewDBStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	testStore(t, store)
}
