package cbytecache

import (
	"reflect"

	"github.com/koykov/cbyte"
)

type arena struct {
	id uint32
	h  reflect.SliceHeader
}

func allocArena(id uint32) *arena {
	a := &arena{id: id}
	a.h = cbyte.InitHeader(0, int(ArenaSize))
	return a
}

func (a *arena) write(b []byte) (n int) {
	n = cbyte.Memcpy(uint64(a.h.Data), uint64(a.h.Len), b)
	a.h.Len += n
	return
}

func (a *arena) read(offset, length uint32) []byte {
	h := reflect.SliceHeader{
		Data: a.h.Data + uintptr(offset),
		Len:  int(length),
		Cap:  int(length),
	}
	return cbyte.Bytes(h)
}

func (a arena) offset() uint32 {
	return uint32(a.h.Len)
}

func (a arena) rest() uint32 {
	return uint32(a.h.Cap - a.h.Len)
}

func (a *arena) release() {
	cbyte.ReleaseHeader(a.h)
}
