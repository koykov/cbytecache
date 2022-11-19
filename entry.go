package cbytecache

import "github.com/koykov/indirect"

type entry struct {
	hash   uint64
	offset uint32
	length uint32
	expire uint32 // overflows at 2106-02-07 06:28:15
	aidptr uintptr
	aptr   uintptr
}

func (e entry) arenaID() uint32 {
	uptr := indirect.ToUnsafePtr(e.aidptr)
	return *(*uint32)(uptr)
}

func (e *entry) arena() *arena {
	uptr := indirect.ToUnsafePtr(e.aptr)
	return (*arena)(uptr)
}
