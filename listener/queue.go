package listener

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/koykov/fastconv"
)

type Enqueuer interface {
	io.Closer
	Enqueue(x interface{}) error
}

type Queue struct {
	Enqueuer Enqueuer
}

func NewQueue(enq Enqueuer) (*Queue, error) {
	if enq == nil {
		return nil, ErrNoEnqueuer
	}
	q := Queue{Enqueuer: enq}
	return &q, nil
}

func (q *Queue) Listen(key string, body []byte) error {
	if q.Enqueuer == nil {
		return ErrNoEnqueuer
	}

	itm := NewItem(len(key) + len(body) + 2)
	itm.Encode(key, body)
	return q.Enqueuer.Enqueue(itm)
}

func (q *Queue) Close() error {
	return q.Enqueuer.Close()
}

type QueueItem []byte

func NewItem(len int) QueueItem {
	itm := make(QueueItem, 0, len)
	return itm
}

func (i *QueueItem) Encode(key string, body []byte) {
	*i = append((*i)[:0], 0)
	*i = append(*i, 0)
	binary.LittleEndian.PutUint16(*i, uint16(len(key)))
	*i = append(*i, key...)
	*i = append(*i, body...)
}

func (i *QueueItem) Decode() (key string, body []byte) {
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
