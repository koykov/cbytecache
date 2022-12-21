package cbytecache

import "fmt"

// MemorySize represents size in bytes.
type MemorySize uint64

// Common sizes.
const (
	Byte     MemorySize = 1
	Kilobyte            = Byte * 1024
	Megabyte            = Kilobyte * 1024
	Gigabyte            = Megabyte * 1024
	Terabyte            = Gigabyte * 1024
	_                   = Terabyte
)

// CacheSize represents memory size types of cache: total, used and free.
type CacheSize struct {
	t, u, f MemorySize
}

// Total returns total size of cache.
func (s CacheSize) Total() MemorySize {
	return s.t
}

// Used returns used size of cache.
func (s CacheSize) Used() MemorySize {
	return s.u
}

// Free returns free size of cache.
func (s CacheSize) Free() MemorySize {
	return s.f
}

// Equal check is x is equal to s.
func (s CacheSize) Equal(x CacheSize) bool {
	return s.t == x.t && s.u == x.u && s.f == x.f
}

// String returns a string representation of size.
func (s CacheSize) String() string {
	return fmt.Sprintf("{total: %d, used: %d, free: %d}", s.t, s.u, s.f)
}
