package cbytecache

import "github.com/koykov/indirect"

type entry struct {
	hash   uint64
	offset uint32
	length uint32
	expire uint32 // overflows at 2106-02-07 06:28:15
	aid    uint32
	lp     uintptr
}

func (e *entry) arena() *arena {
	raw := indirect.ToUnsafePtr(e.lp)
	al := (*arenaQueue)(raw)
	return al.get(int64(e.aid))
}
