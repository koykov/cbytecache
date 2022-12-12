package queue

import (
	"github.com/koykov/cbytecache"
)

// Listener is a cbytecache.Listener implementation that forwards entries to the enqueuer.
type Listener struct {
	enqueuer cbytecache.Enqueuer
}

// NewListener makes new Listener instance with given enqueuer.
func NewListener(enq cbytecache.Enqueuer) (*Listener, error) {
	if enq == nil {
		return nil, cbytecache.ErrNoEnqueuer
	}
	q := Listener{enqueuer: enq}
	return &q, nil
}

func (q *Listener) Listen(entry cbytecache.Entry) error {
	if q.enqueuer == nil {
		return cbytecache.ErrNoEnqueuer
	}
	cpy := entry.Copy()
	return q.enqueuer.Enqueue(cpy)
}

var _ = NewListener
