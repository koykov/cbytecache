package cbytecache

import "math"

const (
	MaxEntrySize = math.MaxUint16
	MaxShardSize = math.MaxUint32

	// todo increase after tests.
	ArenaSize = uint32(Kilobyte)

	cacheStatusNil    = 0
	cacheStatusActive = 1
	cacheStatusClosed = 2

	shardStatusActive  = 0
	shardStatusService = 1
	shardStatusCorrupt = 2
)
