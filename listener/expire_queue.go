package listener

import (
	"github.com/koykov/cbytecache"
)

type ExpireQueue struct {
	Enqueuer cbytecache.Enqueuer
}

func NewQueue(enq cbytecache.Enqueuer) (*ExpireQueue, error) {
	if enq == nil {
		return nil, cbytecache.ErrNoEnqueuer
	}
	q := ExpireQueue{Enqueuer: enq}
	return &q, nil
}

func (q *ExpireQueue) Listen(entry cbytecache.Entry) error {
	if q.Enqueuer == nil {
		return cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return q.Enqueuer.Enqueue(cpy)
}

var _ = NewQueue
