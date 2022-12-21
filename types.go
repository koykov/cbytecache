package cbytecache

import (
	"github.com/koykov/fastconv"
)

/*
 * Exportable types to use in subpackages.
 */

// Enqueuer is the interface that wraps the basic Enqueue method.
//
// Uses for queue.DumpWriter and queue.Listener that contains underlying queue.
type Enqueuer interface {
	Enqueue(interface{}) error
}

// Entry represent external entry object.
type Entry struct {
	Key    string
	Body   []byte
	Expire uint32
}

// Copy copies entry to avoid overwrite entry data.
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

// Size returns entry size in bytes.
func (e Entry) Size() int {
	return len(e.Key) + len(e.Body) + 4
}
