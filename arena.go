package cbytecache

import (
	"reflect"
	"unsafe"

	"github.com/koykov/cbyte"
)

// Memory arena implementation.
type arena struct {
	id   uint32
	h    reflect.SliceHeader
	p, n *arena
}

// Create and alloc space for new arena.
func allocArena(id uint32, ac MemorySize) *arena {
	a := &arena{id: id}
	a.h = cbyte.InitHeader(0, int(ac))
	return a
}

// Get raw unsafe pointer of arena.
//
// Caution! Pointer receiver strongly required here.
func (a *arena) ptr() uintptr {
	uptr := unsafe.Pointer(a)
	return uintptr(uptr)
}

// Write b to arena.
//
// Caution! No bounds check control. External code must guarantee the safety.
func (a *arena) write(b []byte) (n int) {
	// lo, hi := a.h.Len, a.h.Len+len(b)
	// a.h.Len = hi
	// buf := cbyte.Bytes(a.h)
	// n = copy(buf[lo:hi], b)
	// todo check stable
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
	delta := a.h.Cap - a.h.Len
	if delta <= 0 {
		return 0
	}
	return uint32(delta)
}

// Set previous arena.
func (a *arena) setPrev(prev *arena) *arena {
	if prev != nil {
		a.p = prev
	}
	return a
}

// Get previous arena.
func (a *arena) prev() *arena {
	return a.p
}

// Set next arena.
func (a *arena) setNext(next *arena) *arena {
	if next != nil {
		a.n = next
	}
	return a
}

// Get next arena.
func (a *arena) next() *arena {
	return a.n
}

// Reset arena data.
//
// Allocated memory will not release and become available to rewrite.
func (a *arena) reset() {
	a.h.Len = 0
}

// Release memory arena.
//
// Arena object doesn't destroy. Using it afterward is unsafe.
func (a *arena) release() {
	cbyte.ReleaseHeader(a.h)
}
