package cbytecache

import "io"

type DumpWriter interface {
	io.Closer
	Write(item Entry) error
}

type DumpReader interface {
	io.Closer
	Read() (Entry, error)
}
