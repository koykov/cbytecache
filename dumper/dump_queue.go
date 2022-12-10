package dumper

import (
	"github.com/koykov/cbytecache"
)

// DumpQueue is a DumpWriter wrapper over enqueuer.
type DumpQueue struct {
	Enqueuer cbytecache.Enqueuer
}

// NewQueue makes new DumpQueue instance with given enqueuer.
func NewQueue(enq cbytecache.Enqueuer) (*DumpQueue, error) {
	if enq == nil {
		return nil, cbytecache.ErrNoEnqueuer
	}
	q := DumpQueue{Enqueuer: enq}
	return &q, nil
}

func (q *DumpQueue) Write(entry cbytecache.Entry) (int, error) {
	if q.Enqueuer == nil {
		return 0, cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return cpy.Size(), q.Enqueuer.Enqueue(cpy)
}

func (q *DumpQueue) Flush() error {
	return nil
}

var _ = NewQueue
