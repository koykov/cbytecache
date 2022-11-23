package cbytecache

import "time"

type MetricsWriter interface {
	Alloc(bucket string, size uint32)
	Release(bucket string, size uint32)
	Set(bucket string, len uint32, dur time.Duration)
	ArenaAlloc(bucket string, append_ bool)
	ArenaReset(bucket string)
	ArenaFill(bucket string)
	ArenaRelease(bucket string)
	Evict(bucket string, len uint32)
	Miss(bucket string)
	Hit(bucket string, dur time.Duration)
	Expire(bucket string)
	Corrupt(bucket string)
	Collision(bucket string)
	NoSpace(bucket string)
	Dump()
	Load()
}
