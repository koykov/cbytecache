package cbytecache

import "github.com/koykov/indirect"

// Internal entry object.
type entry struct {
	// Hash of entry key. Uses to link entry with index map.
	hash uint64
	// Entry data offset in arena.
	offset uint32
	// Entry length in arena(-s).
	length uint32
	// Expiration timestamp. Will overflow at 2106-02-07 06:28:15
	expire uint32
	// Arena index in queue storage.
	aid uint32
	// Queue raw pointer.
	qp uintptr
}

// Get starting arena contains entry data.
func (e *entry) arena() *arena {
	raw := indirect.ToUnsafePtr(e.qp)
	q := (*arenaQueue)(raw)
	return q.get(int64(e.aid))
}

// Make entry invalid.
func (e *entry) destroy() {
	e.hash = 0
	e.offset, e.length = 0, 0
	e.expire = 0
}

// Check entry is invalid.
//
// Allows to skip processing of previously deleted entries.
func (e *entry) invalid() bool {
	return e.hash == 0 || e.length == 0 || e.expire == 0
}
