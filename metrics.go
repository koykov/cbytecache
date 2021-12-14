package cbytecache

type MetricsWriter interface {
	Alloc(key string, size uint32)
	Free(key string, size uint32)
	Release(key string, size uint32)
	Set(key string, len uint32)
	Evict(key string, len uint32)
	Miss(key string)
	Hit(key string)
	Expire(key string)
	Corrupt(key string)
	Collision(key string)
	NoSpace(key string)
}
