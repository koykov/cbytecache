package cbytecache

type entry struct {
	offset uint32
	length uint16
	expire uint32
}
