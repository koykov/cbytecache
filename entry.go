package cbytecache

type entry struct {
	hash   uint64
	offset uint32
	length uint32
	expire uint32 // overflows at 2106-02-07 06:28:15
	aidptr *uint32
}

func (e entry) arenaID() uint32 {
	return *e.aidptr
}
