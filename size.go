package cbytecache

type MemorySize uint64

const (
	Byte     MemorySize = 1
	Kilobyte            = Byte * 1024
	Megabyte            = Kilobyte * 1024
	Gigabyte            = Megabyte * 1024
	Terabyte            = Gigabyte * 1024
	_                   = Terabyte
)

type CacheSize struct {
	t, u, f MemorySize
}

func (s CacheSize) Total() MemorySize {
	return s.t
}

func (s CacheSize) Used() MemorySize {
	return s.u
}

func (s CacheSize) Free() MemorySize {
	return s.f
}
