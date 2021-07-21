package cbytecache

import (
	"reflect"

	"github.com/koykov/cbyte"
)

type arena reflect.SliceHeader

func (a *arena) bytesRange(offset uint32, length uint16) []byte {
	h := reflect.SliceHeader{
		Data: a.Data + uintptr(offset),
		Len:  int(length),
		Cap:  int(length),
	}
	return cbyte.Bytes(h)
}
