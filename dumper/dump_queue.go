package dumper

import (
	"github.com/koykov/cbytecache"
)

type DumpQueue struct {
	Enqueuer cbytecache.Enqueuer
}

func NewQueue(enq cbytecache.Enqueuer) (*DumpQueue, error) {
	if enq == nil {
		return nil, cbytecache.ErrNoEnqueuer
	}
	q := DumpQueue{Enqueuer: enq}
	return &q, nil
}

func (q *DumpQueue) Write(entry cbytecache.Entry) error {
	if q.Enqueuer == nil {
		return cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return q.Enqueuer.Enqueue(cpy)
}

func (q *DumpQueue) Close() error {
	return q.Enqueuer.Close()
}

var _ = NewQueue
