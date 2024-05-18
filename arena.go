package cbytecache

import (
	"reflect"

	"github.com/koykov/cbyte"
	"github.com/koykov/indirect"
)

// Memory arena implementation.
//
// Arena is minimal block of cache memory. All alloc/reset/fill/release operations can manipulate only the entire arena.
type arena struct {
	id uint32
	h  reflect.SliceHeader
	// Previous/next arenas indices.
	p, n int64
	// Raw queue pointer.
	qp uintptr
}

// Check arena memory is released.
func (a *arena) released() bool {
	return a.h.Data == 0 && a.h.Cap == 0
}

// Check arena memory is empty.
func (a *arena) empty() bool {
	return !a.released() && a.h.Len == 0
}

// Check arena is full.
func (a *arena) full() bool {
	return !a.released() && a.h.Len == a.h.Cap
}

// Write b to arena.
//
// Caution! No bounds check control. External code must guarantee bounds safety.
func (a *arena) write(b []byte) (n int) {
	n = cbyte.Memcpy(uint64(a.h.Data), uint64(a.h.Len), b)
	a.h.Len += n
	return
}

// Read bytes from arena using offset and length.
//
// Caution! No bounds check control. External code must guarantee bounds safety.
func (a *arena) read(offset, length uint32) []byte {
	h := reflect.SliceHeader{
		Data: a.h.Data + uintptr(offset),
		Len:  int(length),
		Cap:  int(length),
	}
	return cbyte.Bytes(h)
}

// Get free space offset (or length of using space).
func (a *arena) offset() uint32 {
	return uint32(a.h.Len)
}

// Get length of free space in arena.
func (a *arena) rest() uint32 {
	delta := a.h.Cap - a.h.Len
	if delta <= 0 {
		return 0
	}
	return uint32(delta)
}

// Set previous arena.
func (a *arena) setPrev(prev *arena) *arena {
	q := a.indirectQueue()
	if q == nil {
		return nil
	}
	a.p = -1
	if prev != nil {
		a.p = int64(prev.id)
	}
	q.buf[a.id] = *a
	return a
}

// Get previous arena.
func (a *arena) prev() *arena {
	if a == nil || a.p == -1 {
		return nil
	}
	q := a.indirectQueue()
	if q == nil {
		return nil
	}
	return &q.buf[a.p]
}

// Set next arena.
func (a *arena) setNext(next *arena) *arena {
	q := a.indirectQueue()
	if q == nil {
		return nil
	}
	a.n = -1
	if next != nil {
		a.n = int64(next.id)
	}
	q.buf[a.id] = *a
	return a
}

// Get next arena.
func (a *arena) next() *arena {
	if a == nil || a.n == -1 {
		return nil
	}
	q := a.indirectQueue()
	if q == nil {
		return nil
	}
	return &q.buf[a.n]
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
	a.h.Data, a.h.Len, a.h.Cap = 0, 0, 0
}

// Indirect queue from raw pointer.
func (a *arena) indirectQueue() *arenaQueue {
	if a.qp == 0 {
		return nil
	}
	raw := indirect.ToUnsafePtr(a.qp)
	return (*arenaQueue)(raw)
}
