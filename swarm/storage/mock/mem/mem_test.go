// Authored and revised by YOC team, 2018
// License placeholder #1

package mem

import (
	"testing"

	"github.com/Yocoin15/Yocoin_Sources/swarm/storage/mock/test"
)

// TestGlobalStore is running test for a GlobalStore
// using test.MockStore function.
func TestGlobalStore(t *testing.T) {
	test.MockStore(t, NewGlobalStore(), 100)
}

// TestImportExport is running tests for importing and
// exporting data between two GlobalStores
// using test.ImportExport function.
func TestImportExport(t *testing.T) {
	test.ImportExport(t, NewGlobalStore(), NewGlobalStore(), 100)
}
