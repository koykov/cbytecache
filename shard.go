package cbytecache

import (
	"reflect"
	"sync"
	"unsafe"
)

type entry struct {
	addr uintptr
	l    int
	e    uint
}

type shard struct {
	service uint32
	mux     sync.RWMutex
	// first variant - simple map
	hmap map[uint64][]byte
	// second variant - pair of slices
	addr    []uint64
	data    [][]byte
	offset  uint64
	lrLimit uint64
	// third variant - entry hmap
	hemap map[uint64]entry
}

func newShard() *shard {
	s := &shard{
		hmap:  make(map[uint64][]byte),
		addr:  make([]uint64, 0),
		data:  make([][]byte, 0),
		hemap: make(map[uint64]entry),
	}
	return s
}

func (s *shard) set0(hash uint64, b []byte) error {
	s.mux.Lock()
	s.hmap[hash] = b
	s.mux.Unlock()
	return nil
}

func (s *shard) get0(hash uint64) ([]byte, error) {
	s.mux.RLock()
	b := s.hmap[hash]
	s.mux.RUnlock()
	return b, nil
}

func (s *shard) set1(hash uint64, b []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	var i uint64
	for i = 0; i < s.offset; i++ {
		if s.addr[i] == hash {
			s.data[i] = b
			return nil
		}
	}
	s.addr = append(s.addr, hash)
	s.data = append(s.data, b)
	s.offset++
	return nil
}

func (s *shard) get1(hash uint64) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var i uint64
	for i = 0; i < s.offset; i++ {
		if s.addr[i] == hash {
			return s.data[i], nil
		}
	}
	return nil, nil
}

func (s *shard) set2(hash uint64, b []byte) error {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var i uint64
	for i = 0; i < s.lrLimit; i += 8 {
		if s.addr[i] == hash {
			s.data[i] = b
			return nil
		}
		if s.addr[i+1] == hash {
			s.data[i+1] = b
			return nil
		}
		if s.addr[i+2] == hash {
			s.data[i+2] = b
			return nil
		}
		if s.addr[i+3] == hash {
			s.data[i+3] = b
			return nil
		}
		if s.addr[i+4] == hash {
			s.data[i+4] = b
			return nil
		}
		if s.addr[i+5] == hash {
			s.data[i+5] = b
			return nil
		}
		if s.addr[i+6] == hash {
			s.data[i+6] = b
			return nil
		}
		if s.addr[i+7] == hash {
			s.data[i+7] = b
			return nil
		}
	}
	for i := s.lrLimit; i < s.offset; i++ {
		if s.addr[i] == hash {
			s.data[i] = b
			return nil
		}
	}
	s.addr = append(s.addr, hash)
	s.data = append(s.data, b)
	s.offset++
	if s.offset%8 == 0 {
		s.lrLimit += 8
	}
	return nil
}

func (s *shard) get2(hash uint64) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var i uint64
	for i = 0; i < s.lrLimit; i += 8 {
		if s.addr[i] == hash {
			return s.data[i], nil
		}
		if s.addr[i+1] == hash {
			return s.data[i+1], nil
		}
		if s.addr[i+2] == hash {
			return s.data[i+2], nil
		}
		if s.addr[i+3] == hash {
			return s.data[i+3], nil
		}
		if s.addr[i+4] == hash {
			return s.data[i+4], nil
		}
		if s.addr[i+5] == hash {
			return s.data[i+5], nil
		}
		if s.addr[i+6] == hash {
			return s.data[i+6], nil
		}
		if s.addr[i+7] == hash {
			return s.data[i+7], nil
		}

	}
	for i = s.lrLimit; i < s.offset; i++ {
		if s.addr[i] == hash {
			return s.data[i], nil
		}
	}
	return nil, nil
}

func (s *shard) set3(hash uint64, b []byte) error {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	s.mux.Lock()
	s.hemap[hash] = entry{
		addr: h.Data,
		l:    h.Len,
		e:    0,
	}
	s.mux.Unlock()
	return nil
}

func (s *shard) get3(hash uint64) ([]byte, error) {
	s.mux.RLock()
	e := s.hemap[hash]
	h := reflect.SliceHeader{
		Data: e.addr,
		Len:  e.l,
		Cap:  e.l,
	}
	s.mux.RUnlock()
	return *(*[]byte)(unsafe.Pointer(&h)), nil
}
