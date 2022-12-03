package cbytecache

import "time"

type MetricsWriter interface {
	Alloc(bucket string, size uint32)
	Fill(bucket string, size uint32)
	Reset(bucket string, size uint32)
	Release(bucket string, size uint32)
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
