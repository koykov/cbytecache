package cbytecache

import "unsafe"

type entry struct {
	offset uint32
	length uint16
	expire uint32 // overflows at 2106-02-07 06:28:15
	aidptr uintptr
}

func (e entry) arenaID() uint32 {
	return *(*uint32)(unsafe.Pointer(e.aidptr))
}
