package cbytecache

import "io"

type Listener interface {
	io.Closer
	Listen(key string, body []byte) error
}
