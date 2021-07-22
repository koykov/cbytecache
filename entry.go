package cbytecache

type entry struct {
	offset uint32
	length uint16
	expire uint32 // overflows at 2106-02-07 06:28:15
}
