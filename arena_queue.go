package cbytecache

import (
	"unsafe"

	"github.com/koykov/cbyte"
)

// Circular queue of bucket's arenas.
type arenaQueue struct {
	// Head, actual and tail arenas indexes pointers.
	// Manipulation of these pointers implements circular queue patterns.
	head_, act_, tail_ int64
	// Arenas list storage.
	buf []arena
}

// Get length of arenas storage.
func (q *arenaQueue) len() int {
	return len(q.buf)
}

// Alloc new arena.
//
// Search in buf for old arena, available to reuse (realloc) or create (alloc) new arena and it to the storage.
func (q *arenaQueue) alloc(prev *arena, size MemorySize) (a *arena) {
	for i := 0; i < len(q.buf); i++ {
		if q.buf[i].released() {
			a = &q.buf[i]
			break
		}
	}
	if a == nil {
		// No arena available on buf, so create new one and add in to the buf.
		a1 := arena{
			id: uint32(q.len()),
			qp: q.ptr(),
			p:  -1,
			n:  -1,
		}
		q.buf = append(q.buf, a1)
		a = &q.buf[q.len()-1]
	}
	// Alloc memory.
	a.h = cbyte.InitHeader(0, int(size))
	// Link prev/new arena.
	a.setPrev(prev)
	if prev != nil {
		prev.setNext(a)
	}
	// Mark new arena as tail.
	q.setTail(a)
	return
}

// Set head arena.
func (q *arenaQueue) setHead(a *arena) *arenaQueue {
	q.head_ = -1
	if a != nil {
		q.head_ = int64(a.id)
	}
	return q
}

// Set actual arena.
func (q *arenaQueue) setAct(a *arena) *arenaQueue {
	q.act_ = -1
	if a != nil {
		q.act_ = int64(a.id)
	}
	return q
}

// Set tail arena.
func (q *arenaQueue) setTail(a *arena) *arenaQueue {
	q.tail_ = -1
	if a != nil {
		q.tail_ = int64(a.id)
	}
	return q
}

// Get head arena.
func (q *arenaQueue) head() *arena {
	return q.get(q.head_)
}

// Get actual arena.
func (q *arenaQueue) act() *arena {
	return q.get(q.act_)
}

// Get tail arena.
func (q *arenaQueue) tail() *arena {
	return q.get(q.tail_)
}

func (q *arenaQueue) get(id int64) *arena {
	if id == -1 {
		return nil
	}
	if id >= int64(len(q.buf)) {
		return nil
	}
	return &q.buf[id]
}

// Recycle arenas using lo as starting arena.
//
// After recycle arenas contains unexpired entries will shift to the head, all other arenas will shift to the end.
func (q *arenaQueue) recycle(lo *arena) {
	if lo == nil {
		return
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

	oh, ot := q.head(), q.tail()
	// Set low+1 as head since it contains at least one unexpired entry.
	q.setHead(lo.next())
	q.head().setPrev(nil)
	// low became new tail.
	q.setTail(lo)
	// Old head previous arena became old tail.
	oh.setPrev(ot)
	// Old tail next arena became old head.
	ot.setNext(oh)
	q.tail().setNext(nil)

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
	//
	// Arenas schema:
	// * [6..9] - potentially empty, but they will reset anyway
	// * [0..2] - contains expired entries, so empty them
	//
	// Recycle end.
}

// Get raw unsafe pointer of arena list.
//
// Caution! Pointer receiver strongly required here.
func (q *arenaQueue) ptr() uintptr {
	uptr := unsafe.Pointer(q)
	return uintptr(uptr)
}
