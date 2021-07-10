package cbytecache

import (
	"sync"
)

type shard struct {
	mux   sync.RWMutex
	items map[uint32]entry
}

func newShard() *shard {
	s := &shard{
		items: make(map[uint32]entry),
	}
	return s
}

func (s *shard) set(hash uint32, b []byte) error {
	s.mux.Lock()
	// ...
	s.mux.Unlock()
	return nil
}

func (s *shard) get(dst []byte, hash uint32) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	// ...
	return dst, ErrNotFound
}
