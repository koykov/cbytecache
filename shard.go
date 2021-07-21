package cbytecache

import (
	"sync"
	"sync/atomic"
)

type shard struct {
	mux    sync.RWMutex
	index  map[uint64]uint32
	entry  []entry
	arena  []arena
	nowPtr *uint32
}

func newShard(nowPtr *uint32) *shard {
	s := &shard{
		index:  make(map[uint64]uint32),
		nowPtr: nowPtr,
	}
	return s
}

func (s *shard) set(hash uint64, b []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	// ...
	return ErrOK
}

func (s *shard) get(dst []byte, hash uint64) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var (
		idx uint32
		ok  bool
	)
	if idx, ok = s.index[hash]; !ok {
		return dst, ErrNotFound
	}
	if idx >= uint32(len(s.entry)) {
		return dst, ErrNotFound
	}
	entry := s.entry[idx]
	if entry.expire < s.now() {
		return dst, ErrNotFound
	}

	arenaIdx := entry.offset / ArenaSize
	arenaOffset := entry.offset % ArenaSize

	if arenaIdx >= uint32(len(s.arena)) {
		return dst, ErrNotFound
	}
	arena := &s.arena[arenaIdx]

	if ArenaSize-arenaOffset < uint32(entry.length) {
		dst = append(dst, arena.bytesRange(arenaOffset, entry.length)...)
	} else {
		// todo implement worst case (entry shared among arenas).
	}

	return dst, ErrOK
}

func (s *shard) now() uint32 {
	return atomic.LoadUint32(s.nowPtr)
}
