package dumper

import (
	"github.com/koykov/cbytecache"
)

// Queue is a DumpWriter implementation that forwards entries to the enqueuer.
type Queue struct {
	Enqueuer cbytecache.Enqueuer
}

// NewQueue makes new Queue instance with given enqueuer.
func NewQueue(enq cbytecache.Enqueuer) (*Queue, error) {
	if enq == nil {
		return nil, cbytecache.ErrNoEnqueuer
	}
	q := Queue{Enqueuer: enq}
	return &q, nil
}

func (q *Queue) Write(entry cbytecache.Entry) (int, error) {
	if q.Enqueuer == nil {
		return 0, cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return cpy.Size(), q.Enqueuer.Enqueue(cpy)
}

func (q *Queue) Flush() error {
	return nil
}

var _ = NewQueue
