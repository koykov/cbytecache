package cbytecache

import "time"

type MetricsWriter interface {
	Alloc(bucket string)
	Fill(bucket string)
	Reset(bucket string)
	Release(bucket string)
	ArenaMap(bucket string, total, used, free, size uint32)
	Set(bucket string, dur time.Duration)
	Hit(bucket string, dur time.Duration)
	Evict(bucket string)
	Miss(bucket string)
	Expire(bucket string)
	Corrupt(bucket string)
	Collision(bucket string)
	NoSpace(bucket string)
	Dump()
	Load()
}
