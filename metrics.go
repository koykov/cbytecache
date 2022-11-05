package cbytecache

import "time"

type MetricsWriter interface {
	Alloc(size uint32)
	Free(size uint32)
	Release(size uint32)
	Set(len uint32, dur time.Duration)
	Evict()
	Miss()
	Hit(dur time.Duration)
	Expire()
	Corrupt()
	Collision()
	NoSpace()
	Dump()
	Load()
}
