package cbytecache

import (
	"math"
	"time"
)

const (
	MaxEntrySize  = math.MaxUint16
	MaxBucketSize = math.MaxUint32

	MinExpireInterval = time.Second

	// todo increase after tests.
	ArenaSize = uint32(Kilobyte)

	cacheStatusNil    = 0
	cacheStatusActive = 1
	cacheStatusClosed = 2

	bucketStatusActive  = 0
	bucketStatusService = 1
	bucketStatusCorrupt = 2

	evictWorkers = 16
)
