package cbytecache

type MemorySize uint64

const (
	Byte     MemorySize = 1
	Kilobyte            = Byte * 1024
	Megabyte            = Kilobyte * 1024
	Gigabyte            = Megabyte * 1024
	Terabyte            = Gigabyte * 1024
)
