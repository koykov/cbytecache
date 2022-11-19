package cbytecache

import (
	"unsafe"

	"github.com/koykov/cbyte"
	"github.com/koykov/indirect"
)

type arenaList struct {
	head_, act_, tail_ uintptr

	i, l, c uint32

	buf []*arena
}

func (l *arenaList) len() int {
	return len(l.buf)
}

func (l *arenaList) alloc(prev *arena, size MemorySize) *arena {
	var a *arena
	for i := 0; i < len(l.buf); i++ {
		if l.buf[i].empty() {
			a = l.buf[i]
			break
		}
	}
	if a == nil {
		a = &arena{id: uint32(l.len())}
		l.buf = append(l.buf, a)
	}
	a.h = cbyte.InitHeader(0, int(size))
	a.setPrev(prev)
	if prev != nil {
		prev.setNext(a)
	}
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

func (l *arenaList) recycle(lo *arena) int {
	if lo == nil {
		return 0
	}
	head, tail := l.head(), l.tail()
	l.setHead(lo.n)
	l.head().setPrev(nil)
	l.setTail(lo)
	head.setPrev(tail)
	tail.setNext(head)
	l.tail().setNext(nil)

	var c int
	a := head
	for a != nil {
		a.reset()
		a = a.n
		c++
	}
	return c
}
