package cbytecache

import (
	"unsafe"

	"github.com/koykov/cbyte"
	"github.com/koykov/indirect"
)

// Container of bucket's arenas.
type arenaList struct {
	// Head, actual and tail arenas pointers.
	head_, act_, tail_ uintptr
	// Arenas list storage.
	buf []*arena
}

// Get length of arenas storage.
func (l *arenaList) len() int {
	return len(l.buf)
}

// Alloc new arena.
//
// Search in buf old arena, available to reuse (realloc) or create (alloc) new arena and it to the storage.
func (l *arenaList) alloc(prev *arena, size MemorySize) (a *arena, ok bool) {
	for i := 0; i < len(l.buf); i++ {
		if l.buf[i].released() {
			a = l.buf[i]
			break
		}
	}
	if a == nil {
		a = &arena{id: uint32(l.len())}
		l.buf = append(l.buf, a)
		ok = true
	}
	a.h = cbyte.InitHeader(0, int(size))
	a.setPrev(prev)
	if prev != nil {
		prev.setNext(a)
	}
	l.setTail(a)
	return
}

// Set head arena.
func (l *arenaList) setHead(head *arena) *arenaList {
	l.head_ = uintptr(unsafe.Pointer(head))
	return l
}

// Set actual arena.
func (l *arenaList) setAct(act *arena) *arenaList {
	l.act_ = uintptr(unsafe.Pointer(act))
	return l
}

// Set tail arena.
func (l *arenaList) setTail(tail *arena) *arenaList {
	l.tail_ = uintptr(unsafe.Pointer(tail))
	return l
}

// Get head arena.
func (l *arenaList) head() *arena {
	raw := indirect.ToUnsafePtr(l.head_)
	return (*arena)(raw)
}

// Get actual arena.
func (l *arenaList) act() *arena {
	raw := indirect.ToUnsafePtr(l.act_)
	return (*arena)(raw)
}

// Get tail arena.
func (l *arenaList) tail() *arena {
	raw := indirect.ToUnsafePtr(l.tail_)
	return (*arena)(raw)
}

// Recycle arenas using lo as starting arena.
//
// After recycle arenas contains unexpired entries will shift to the head, all other arenas will shift to the end.
func (l *arenaList) recycle(lo *arena) int {
	if lo == nil {
		return 0
	}

	// Old arena sequence:
	// ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐
	// │ 0 │ 1 │ 2 │ 3 │ 4 │ 5 │ 6 │ 7 │ 8 │ 9 │
	// └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘
	//   ▲       ▲           ▲               ▲
	//   │       │           │               │
	//   head    │           │               │
	//   low ────┘           │               │
	//   actual ─────────────┘               │
	//   tail ───────────────────────────────┘
	// low is a last arena contains only expired entries.

	oh, ot := l.head(), l.tail()
	// Set low+1 as head since it contains at least one unexpired entry.
	l.setHead(lo.next())
	l.head().setPrev(nil)
	// low became new tail.
	l.setTail(lo)
	// Old head previous arena became old tail.
	oh.setPrev(ot)
	// Old tail next arena became old head.
	ot.setNext(oh)
	l.tail().setNext(nil)

	// New sequence:
	// ┌───┬───┬───┬───┬───┬───┬───┬───┬───┬───┐
	// │ 3 │ 4 │ 5 │ 6 │ 7 │ 8 │ 9 │ 0 │ 1 │ 2 │
	// └───┴───┴───┴───┴───┴───┴───┴───┴───┴───┘
	//   ▲       ▲                           ▲
	//   │       │                           │
	//   head    │                           │
	//   actual ─┘                           │
	//   tail ───────────────────────────────┘
	// Arena #3 contains at least one unexpired entry, so it became new head.
	// Arena #2 became new tail.
	// Recycle end.

	// Clear all arenas after actual:
	// * [6..9] - potentially empty, but reset them anyway
	// * [0..2] - contains expired entries, so empty them
	var c int
	a := l.act().next()
	for a != nil {
		if !a.empty() {
			a.reset()
			c++
		}
		a = a.next()
	}
	return c
}
