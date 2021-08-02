package cbytecache

import (
	"math"
	"time"
)

const (
	MaxEntrySize = math.MaxUint16
	MaxShardSize = math.MaxUint32

	MinExpireInterval = time.Second

	// todo increase after tests.
	ArenaSize = uint32(Kilobyte)

	cacheStatusNil    = 0
	cacheStatusActive = 1
	cacheStatusClosed = 2

	shardStatusActive  = 0
	shardStatusService = 1
	shardStatusCorrupt = 2

	evictWorkers = 16
)
