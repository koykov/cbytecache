package cbytecache

import "io"

type Dumper interface {
	io.Closer
	Dump(key string, body []byte, expire uint32) error
}
