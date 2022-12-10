package listener

import (
	"github.com/koykov/cbytecache"
)

// Queue is a Listener implementation that forwards entries to the enqueuer.
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

func (q *Queue) Listen(entry cbytecache.Entry) error {
	if q.Enqueuer == nil {
		return cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return q.Enqueuer.Enqueue(cpy)
}

var _ = NewQueue
