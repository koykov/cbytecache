package cbytecache

import (
	"math"
	"time"
)

const (
	MaxBucketSize = math.MaxUint32

	MinExpireInterval = time.Second

	ArenaSize = uint32(16 * Megabyte)

	cacheStatusNil    = 0
	cacheStatusActive = 1
	cacheStatusClosed = 2

	bucketStatusActive  = 0
	bucketStatusService = 1
	bucketStatusCorrupt = 2

	resetWorkers = 16
	evictWorkers = 16
)
