package cbytecache

import (
	"sync/atomic"
	"unsafe"

	"github.com/koykov/cbyte"
	"github.com/koykov/indirect"
)

type arenaList struct {
	head_, act_, tail_ uintptr

	i, l, c uint32

	buf []*arena
}

func (l *arenaList) alloc(prev *arena, size MemorySize) *arena {
	a := &arena{id: atomic.AddUint32(&l.i, 1)}
	a.h = cbyte.InitHeader(0, int(size))
	a.setPrev(prev)
	prev.setNext(a)
	l.buf = append(l.buf, a)
	l.setTail(a)
	return a
}

func (l *arenaList) setHead(head *arena) *arenaList {
	l.head_ = uintptr(unsafe.Pointer(head))
	return l
}

func (l *arenaList) setAct(act *arena) *arenaList {
	l.act_ = uintptr(unsafe.Pointer(act))
	return l
}

func (l *arenaList) setTail(tail *arena) *arenaList {
	l.tail_ = uintptr(unsafe.Pointer(tail))
	return l
}

func (l *arenaList) head() *arena {
	raw := indirect.ToUnsafePtr(l.head_)
	return (*arena)(raw)
}

func (l *arenaList) act() *arena {
	raw := indirect.ToUnsafePtr(l.act_)
	return (*arena)(raw)
}

func (l *arenaList) tail() *arena {
	raw := indirect.ToUnsafePtr(l.tail_)
	return (*arena)(raw)
}
