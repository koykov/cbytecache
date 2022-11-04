package dumper

import (
	"errors"
	"io"

	"github.com/koykov/cbytecache"
)

/*
 * Enqueuer group.
 */

type Enqueuer interface {
	io.Closer
	Enqueue(x interface{}) error
}

/*
 * DumpQueue group.
 */

type DumpQueue struct {
	Enqueuer Enqueuer
}

func NewQueue(enq Enqueuer) (*DumpQueue, error) {
	if enq == nil {
		return nil, ErrNoEnqueuer
	}
	q := DumpQueue{Enqueuer: enq}
	return &q, nil
}

func (q *DumpQueue) Write(item cbytecache.DumpItem) error {
	if q.Enqueuer == nil {
		return ErrNoEnqueuer
	}

	return q.Enqueuer.Enqueue(item)
}

func (q *DumpQueue) Close() error {
	return q.Enqueuer.Close()
}

var (
	ErrNoEnqueuer = errors.New("no enqueuer provided")

	_ = NewQueue
)
