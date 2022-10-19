package cbytecache

type MetricsWriter interface {
	Alloc(size uint32)
	Free(size uint32)
	Release(size uint32)
	Set(len uint32)
	Evict(len uint32)
	Miss()
	Hit()
	Expire()
	Corrupt()
	Collision()
	NoSpace()
}
