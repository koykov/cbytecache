package cbytecache

import "sync"

type shard struct {
	service uint32
	mux     sync.RWMutex
	// first variant - simple map
	hmap map[uint64][]byte
	// second variant - pair of slices
	addr   []uint64
	data   [][]byte
	offset uint64
	// third variant - ?
	// ...
}

func newShard() *shard {
	s := &shard{
		hmap: make(map[uint64][]byte),
		addr: make([]uint64, 0),
		data: make([][]byte, 0),
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
