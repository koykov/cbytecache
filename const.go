package cbytecache

import "math"

const (
	MaxEntrySize = math.MaxUint16
	MaxShardSize = math.MaxUint32

	// todo increase after tests.
	ArenaSize = uint32(Kilobyte)
)
