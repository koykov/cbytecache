package cbytecache

import (
	"sync"

	"github.com/koykov/byteptr"
	"github.com/koykov/cbyte"
)

type shard struct {
	mux   sync.RWMutex
	items map[uint64]byteptr.Byteptr
	h64   cbyte.SliceHeader64
}

func newShard() *shard {
	s := &shard{
		items: make(map[uint64]byteptr.Byteptr),
	}
	s.h64.Data = uintptr(cbyte.Init64(uint64(1 * Megabyte)))
	return s
}

func (s *shard) set(hash uint64, b []byte) error {
	s.mux.Lock()
	if s.h64.Cap-s.h64.Len < uint64(len(b)) {
		// todo grow me
	}
	bptr := byteptr.Byteptr{} // todo implement Byteptr64
	bptr.SetOffset(int(s.h64.Len)).SetLen(len(b))
	s.items[hash] = bptr
	cbyte.Memcpy(uint64(s.h64.Data), s.h64.Len, b)
	s.mux.Unlock()
	return nil
}

func (s *shard) get(hash uint64) ([]byte, error) {
	s.mux.RLock()
	if bptr, ok := s.items[hash]; ok {
		return bptr.Bytes(), nil
	}
	s.mux.RUnlock()
	return nil, ErrNotFound
}
