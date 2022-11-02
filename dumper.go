package cbytecache

import "io"

type DumpItem struct {
	Key    string
	body   []byte
	expire uint32
}

type DumpWriter interface {
	io.Closer
	Write(item DumpItem) error
}

type DumpReader interface {
	io.Closer
	Read() (DumpItem, error)
}
