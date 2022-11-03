package cbytecache

import (
	"math"
	"time"
)

const (
	MaxBucketSize = math.MaxUint32
	MaxKeySize    = math.MaxUint16

	MinExpireInterval = time.Second

	defaultArenaCapacity = Megabyte

	keySizeBytes = 2

	cacheStatusNil    = 0
	cacheStatusActive = 1
	cacheStatusClosed = 2

	bucketStatusActive  = 0
	bucketStatusService = 1
	bucketStatusCorrupt = 2

	defaultResetWorkers     = 16
	defaultReleaseWorkers   = 16
	defaultExpireWorkers    = 16
	defaultVacuumWorkers    = 16
	defaultDumpWriteWorkers = 16
	defaultDumpReadWorkers  = 16
)
