package cbytecache

import (
	"reflect"

	"github.com/koykov/cbyte"
)

// Memory arena implementation.
type arena struct {
	id uint32
	h  reflect.SliceHeader
}

// Create and alloc space for new arena.
func allocArena(id uint32) *arena {
	a := &arena{id: id}
	a.h = cbyte.InitHeader(0, int(ArenaSize))
	return a
}

// Write b to arena.
//
// Caution! No bounds check control. External code must guarantee the safety.
func (a *arena) write(b []byte) (n int) {
	n = cbyte.Memcpy(uint64(a.h.Data), uint64(a.h.Len), b)
	a.h.Len += n
	return
}

// Read bytes from arena using offset and length.
//
// Caution! No bounds check control. External code must guarantee the safety.
func (a *arena) read(offset, length uint32) []byte {
	h := reflect.SliceHeader{
		Data: a.h.Data + uintptr(offset),
		Len:  int(length),
		Cap:  int(length),
	}
	return cbyte.Bytes(h)
}

// Get free space offset (or length of using space).
func (a arena) offset() uint32 {
	return uint32(a.h.Len)
}

// Get length of free space in arena.
func (a arena) rest() uint32 {
	return uint32(a.h.Cap - a.h.Len)
}

// Release memory arena.
//
// Arena object doesn't destroy. Using it afterward is unsafe.
func (a *arena) release() {
	cbyte.ReleaseHeader(a.h)
}
