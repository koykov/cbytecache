package cbytecache

import (
	"reflect"
)

type arena struct {
	payload reflect.SliceHeader
	next    *arena
}
