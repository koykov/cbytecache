package cbytecache

import (
	"reflect"

	"github.com/koykov/cbyte"
)

type arena struct {
	id uint32
	h  reflect.SliceHeader
}

func (a *arena) bytesRange(offset, length uint32) []byte {
	h := reflect.SliceHeader{
		Data: a.h.Data + uintptr(offset),
		Len:  int(length),
		Cap:  int(length),
	}
	return cbyte.Bytes(h)
}
