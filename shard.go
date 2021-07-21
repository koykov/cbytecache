package cbytecache

import (
	"sync"
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
	// ...
	s.mux.Unlock()
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

	// ...
	_ = idx

	return dst, ErrNotFound
}
