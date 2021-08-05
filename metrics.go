package cbytecache

type MetricsWriter interface {
	Grow(size uint32)
	Reduce(size uint32)
	Release(size uint32)
	Set(len uint16)
	Evict(len uint16)
	Miss()
	Hit()
	Expire()
	Corrupt()
}
