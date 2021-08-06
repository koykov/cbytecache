package cbytecache

type MetricsWriter interface {
	Alloc(size uint32)
	Free(size uint32)
	Release(size uint32)
	Set(len uint16)
	Evict(len uint16)
	Miss()
	Hit()
	Expire()
	Corrupt()
	NoSpace()
}
