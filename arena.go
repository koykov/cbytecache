package cbytecache

import (
	"reflect"

	"github.com/koykov/cbyte"
)

type arena struct {
	id, offset uint32

	h reflect.SliceHeader
}

func allocArena(id uint32) *arena {
	a := &arena{id: id}
	a.h = cbyte.InitHeader(0, int(ArenaSize))
	return a
}

func (a *arena) bytesCopy(offset uint32, b []byte) {
	n := cbyte.Memcpy(uint64(a.h.Data), uint64(offset), b)
	a.h.Len += n
}

func (a *arena) bytesRange(offset, length uint32) []byte {
	h := reflect.SliceHeader{
		Data: a.h.Data + uintptr(offset),
		Len:  int(length),
		Cap:  int(length),
	}
	return cbyte.Bytes(h)
}
