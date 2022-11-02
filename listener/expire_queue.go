package listener

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/koykov/fastconv"
)

/*
 * Enqueuer group.
 */

type Enqueuer interface {
	io.Closer
	Enqueue(x interface{}) error
}

/*
 * ExpireQueue group.
 */

type ExpireQueue struct {
	Enqueuer Enqueuer
}

func NewQueue(enq Enqueuer) (*ExpireQueue, error) {
	if enq == nil {
		return nil, ErrNoEnqueuer
	}
	q := ExpireQueue{Enqueuer: enq}
	return &q, nil
}

func (q *ExpireQueue) Listen(key string, body []byte) error {
	if q.Enqueuer == nil {
		return ErrNoEnqueuer
	}

	itm := NewItem(len(key) + len(body) + 2)
	itm.Encode(key, body)
	return q.Enqueuer.Enqueue(itm)
}

func (q *ExpireQueue) Close() error {
	return q.Enqueuer.Close()
}

/*
 * ExpireQueueItem group.
 */

type ExpireQueueItem []byte

func NewItem(len int) ExpireQueueItem {
	itm := make(ExpireQueueItem, 0, len)
	return itm
}

func (i *ExpireQueueItem) Encode(key string, body []byte) {
	*i = append((*i)[:0], 0)
	*i = append(*i, 0)
	binary.LittleEndian.PutUint16(*i, uint16(len(key)))
	*i = append(*i, key...)
	*i = append(*i, body...)
}

func (i *ExpireQueueItem) Decode() (key string, body []byte) {
	if len(*i) < 2 {
		return
	}
	l := binary.LittleEndian.Uint16((*i)[:2])
	if len(*i)-2 < int(l) {
		return
	}
	key = fastconv.B2S((*i)[2 : l+2])
	body = (*i)[l+2:]
	return
}

var (
	ErrNoEnqueuer = errors.New("no enqueuer provided")

	_ = NewQueue
)
