package queue

import (
	"github.com/koykov/cbytecache"
)

// DumpWriter is a cbytecache.DumpWriter implementation that forwards entries to the enqueuer.
type DumpWriter struct {
	enqueuer cbytecache.Enqueuer
}

// NewDumpWriter makes new DumpWriter instance with given enqueuer.
func NewDumpWriter(enq cbytecache.Enqueuer) (*DumpWriter, error) {
	if enq == nil {
		return nil, cbytecache.ErrNoEnqueuer
	}
	q := DumpWriter{enqueuer: enq}
	return &q, nil
}

func (q *DumpWriter) Write(entry cbytecache.Entry) (int, error) {
	if q.enqueuer == nil {
		return 0, cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return cpy.Size(), q.enqueuer.Enqueue(cpy)
}

func (q *DumpWriter) Flush() error {
	return nil
}

var _ = NewDumpWriter
