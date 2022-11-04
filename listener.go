package cbytecache

import "io"

type Listener interface {
	io.Closer
	Listen(entry Entry) error
}
