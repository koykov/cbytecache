package cbytecache

import (
	"github.com/koykov/fastconv"
)

/*
 * Exportable types to use in subpackages.
 */

type Enqueuer interface {
	Enqueue(interface{}) error
}

type Entry struct {
	Key    string
	Body   []byte
	Expire uint32
}

func (e Entry) Copy() Entry {
	buf := make([]byte, 0, len(e.Key)+len(e.Body))
	buf = append(buf, e.Key...)
	buf = append(buf, e.Body...)
	var cpy Entry
	cpy.Key = fastconv.B2S(buf[:len(e.Key)])
	cpy.Body = buf[len(e.Key):]
	cpy.Expire = e.Expire
	return cpy
}
