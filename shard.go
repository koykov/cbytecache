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
	config *Config
	nowPtr *uint32
}

func newShard(nowPtr *uint32, config *Config) *shard {
	s := &shard{
		index:  make(map[uint64]uint32),
		config: config,
		nowPtr: nowPtr,
	}
	return s
}

func (s *shard) set(h uint64, b []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	// ...
	return ErrOK
}

func (s *shard) get(dst []byte, h uint64) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var (
		idx uint32
		ok  bool
	)
	if idx, ok = s.index[h]; !ok {
		s.m().Miss()
		return dst, ErrNotFound
	}
	if idx >= uint32(len(s.entry)) {
		s.m().Miss()
		return dst, ErrNotFound
	}
	entry := s.entry[idx]
	if entry.expire < s.now() {
		s.m().HitExpired()
		return dst, ErrNotFound
	}

	arenaIdx := entry.offset / ArenaSize
	arenaOffset := entry.offset % ArenaSize

	if arenaIdx >= s.alen() {
		s.m().Miss()
		return dst, ErrNotFound
	}
	arena := &s.arena[arenaIdx]

	arenaRest := ArenaSize - arenaOffset
	if arenaRest < uint32(entry.length) {
		dst = append(dst, arena.bytesRange(arenaOffset, uint32(entry.length))...)
	} else {
		// todo test me.
		rest := uint32(entry.length)
	loop:
		dst = append(dst, arena.bytesRange(arenaOffset, arenaRest)...)
		rest -= arenaRest
		arenaIdx++
		if arenaIdx >= s.alen() {
			s.m().HitCorrupted()
			return dst, ErrEntryCorrupt
		}
		arena = &s.arena[arenaIdx]
		arenaOffset = 0
		arenaRest = min(rest, ArenaSize)
		if rest > 0 {
			goto loop
		}
	}

	s.m().HitOK()
	return dst, ErrOK
}

func (s *shard) now() uint32 {
	return atomic.LoadUint32(s.nowPtr)
}

func (s *shard) alen() uint32 {
	return uint32(len(s.arena))
}

func (s *shard) m() MetricsWriter {
	return s.config.MetricsWriter
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
