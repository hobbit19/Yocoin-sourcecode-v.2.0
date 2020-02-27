// Authored and revised by YOC team, 2018
// License placeholder #1

package mru

import (
	"bytes"
	"context"
	"time"

	"github.com/Yocoin15/Yocoin_Sources/swarm/storage"
)

const (
	defaultStoreTimeout    = 4000 * time.Millisecond
	hasherCount            = 8
	resourceHashAlgorithm  = storage.SHA3Hash
	defaultRetrieveTimeout = 100 * time.Millisecond
)

// resource caches resource data and the metadata of its root chunk.
type resource struct {
	resourceUpdate
	ResourceMetadata
	*bytes.Reader
	lastKey storage.Address
	updated time.Time
}

func (r *resource) Context() context.Context {
	return context.TODO()
}

// TODO Expire content after a defined period (to force resync)
func (r *resource) isSynced() bool {
	return !r.updated.IsZero()
}

// implements storage.LazySectionReader
func (r *resource) Size(ctx context.Context, _ chan bool) (int64, error) {
	if !r.isSynced() {
		return 0, NewError(ErrNotSynced, "Not synced")
	}
	return int64(len(r.resourceUpdate.data)), nil
}

//returns the resource's human-readable name
func (r *resource) Name() string {
	return r.ResourceMetadata.Name
}

// Helper function to calculate the next update period number from the current time, start time and frequency
func getNextPeriod(start uint64, current uint64, frequency uint64) (uint32, error) {
	if current < start {
		return 0, NewErrorf(ErrInvalidValue, "given current time value %d < start time %d", current, start)
	}
	if frequency == 0 {
		return 0, NewError(ErrInvalidValue, "frequency is 0")
	}
	timeDiff := current - start
	period := timeDiff / frequency
	return uint32(period + 1), nil
}
