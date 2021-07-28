package cbytecache

import (
	"sync"
	"sync/atomic"

	"github.com/koykov/cbytebuf"
)

type shard struct {
	config *Config
	status uint32
	nowPtr *uint32

	mux   sync.RWMutex
	buf   *cbytebuf.CByteBuf
	index map[uint64]uint32
	entry []entry
	arena []arena

	arenaOffset uint32
}

func newShard(nowPtr *uint32, config *Config) *shard {
	s := &shard{
		config: config,
		buf:    cbytebuf.NewCByteBuf(),
		index:  make(map[uint64]uint32),
		nowPtr: nowPtr,
	}
	return s
}

func (s *shard) set(h uint64, b []byte) (err error) {
	if err = s.checkStatus(); err != nil {
		return
	}

	s.mux.Lock()
	err = s.setLF(h, b)
	s.mux.Unlock()
	return
}

func (s *shard) setm(h uint64, m MarshallerTo) (err error) {
	if err = s.checkStatus(); err != nil {
		return
	}

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
	// Look for existing entry to reset it.
	var e *entry
	if idx, ok := s.index[h]; ok {
		if idx < s.elen() {
			e = &s.entry[idx]
		}
	}
	if e != nil {
		e.hash = 0
	}

	if s.arenaOffset >= s.alen() {
	alloc1:
		arena := allocArena(s.alen())
		s.arena = append(s.arena, *arena)
		if s.alen() <= s.arenaOffset {
			goto alloc1
		}
	}
	arena := &s.arena[s.arenaOffset]
	arenaID := &arena.id
	arenaOffset := arena.offset
	arenaRest := ArenaSize - arena.offset
	rest := uint32(len(b))
	if arenaRest >= rest {
		arena.bytesCopy(arena.offset, b)
		arena.offset += rest
	} else {
		// todo test me.
	loop:
		arena.bytesCopy(arena.offset, b[:arenaRest])
		rest -= arenaRest
		s.arenaOffset++
		if s.arenaOffset >= s.alen() {
		alloc2:
			arena := allocArena(s.alen())
			s.arena = append(s.arena, *arena)
			if s.alen() <= s.arenaOffset {
				goto alloc2
			}
		}
		arena = &s.arena[s.arenaOffset]
		arenaRest = min(rest, ArenaSize)
		if rest > 0 {
			goto loop
		}
	}

	s.entry = append(s.entry, entry{
		hash:   h,
		offset: arenaOffset,
		length: uint16(len(b)),
		expire: s.now() + uint32(s.config.Expire)/1e9,
		aidptr: arenaID,
	})
	s.index[h] = s.elen() - 1

	s.m().Set(len(b))
	return ErrOK
}

func (s *shard) get(dst []byte, h uint64) ([]byte, error) {
	if err := s.checkStatus(); err != nil {
		return dst, err
	}

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
	if idx >= s.elen() {
		s.m().Miss()
		return dst, ErrNotFound
	}
	entry := &s.entry[idx]
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
	if entry.offset+uint32(entry.length) < ArenaSize {
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

func (s *shard) checkStatus() error {
	if status := atomic.LoadUint32(&s.status); status != shardStatusActive {
		if status == shardStatusService {
			return ErrShardService
		}
		// todo check corrupted status.
	}
	return nil
}

func (s *shard) now() uint32 {
	return atomic.LoadUint32(s.nowPtr)
}

func (s *shard) alen() uint32 {
	return uint32(len(s.arena))
}

func (s *shard) elen() uint32 {
	return uint32(len(s.entry))
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
