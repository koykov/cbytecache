package cbytecache

import (
	"sync"
	"sync/atomic"

	"github.com/koykov/cbytebuf"
)

type shard struct {
	mux    sync.RWMutex
	buf    *cbytebuf.CByteBuf
	index  map[uint64]uint32
	entry  []entry
	arena  []arena
	config *Config
	nowPtr *uint32
}

func newShard(nowPtr *uint32, config *Config) *shard {
	s := &shard{
		buf:    cbytebuf.NewCByteBuf(),
		index:  make(map[uint64]uint32),
		config: config,
		nowPtr: nowPtr,
	}
	return s
}

func (s *shard) set(h uint64, b []byte) (err error) {
	s.mux.Lock()
	err = s.setLF(h, b)
	s.mux.Unlock()
	return
}

func (s *shard) setm(h uint64, m MarshallerTo) (err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.buf.ResetLen()
	if _, err = s.buf.WriteMarshallerTo(m); err != nil {
		return
	}
	err = s.setLF(h, s.buf.Bytes())
	return
}

func (s *shard) setLF(h uint64, b []byte) error {
	_ = h
	s.m().Set(len(b))
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

	// arenaIdx := entry.offset / ArenaSize
	// arenaOffset := entry.offset % ArenaSize
	arenaID := entry.arenaID()
	arenaOffset := entry.offset

	if arenaID >= s.alen() {
		s.m().Miss()
		return dst, ErrNotFound
	}
	arena := &s.arena[arenaID]

	arenaRest := ArenaSize - arenaOffset
	if arenaRest < uint32(entry.length) {
		dst = append(dst, arena.bytesRange(arenaOffset, uint32(entry.length))...)
	} else {
		// todo test me.
		rest := uint32(entry.length)
	loop:
		dst = append(dst, arena.bytesRange(arenaOffset, arenaRest)...)
		rest -= arenaRest
		arenaID++
		if arenaID >= s.alen() {
			s.m().HitCorrupted()
			return dst, ErrEntryCorrupt
		}
		arena = &s.arena[arenaID]
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
