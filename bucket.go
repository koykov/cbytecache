package cbytecache

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/koykov/cbytebuf"
)

type bucket struct {
	config  *Config
	status  uint32
	maxSize uint32
	nowPtr  *uint32

	mux   sync.RWMutex
	buf   *cbytebuf.CByteBuf
	index map[uint64]uint32
	entry []entry

	arena, arenaBuf []arena

	arenaOffset uint32
}

func (s *bucket) set(h uint64, b []byte) (err error) {
	if err = s.checkStatus(); err != nil {
		return
	}

	s.mux.Lock()
	err = s.setLF(h, b)
	s.mux.Unlock()
	return
}

func (s *bucket) setm(h uint64, m MarshallerTo) (err error) {
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

func (s *bucket) setLF(h uint64, b []byte) error {
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

	blen := uint16(len(b))

	if s.arenaOffset >= s.alen() {
		if s.alen()*ArenaSize+ArenaSize > s.maxSize {
			s.m().NoSpace()
			return ErrNoSpace
		}
	alloc1:
		s.m().Alloc(ArenaSize)
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
		b = b[arenaRest:]
		rest -= arenaRest
		s.arenaOffset++
		if s.arenaOffset >= s.alen() {
			if s.alen()*ArenaSize+ArenaSize > s.maxSize {
				return ErrNoSpace
			}
		alloc2:
			s.m().Alloc(ArenaSize)
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
		length: blen,
		expire: s.now() + uint32(s.config.Expire)/1e9,
		aidptr: arenaID,
	})
	s.index[h] = s.elen() - 1

	s.m().Set(blen)
	return ErrOK
}

func (s *bucket) get(dst []byte, h uint64) ([]byte, error) {
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
		s.m().Expire()
		return dst, ErrNotFound
	}

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
			s.m().Corrupt()
			return dst, ErrEntryCorrupt
		}
		arena = &s.arena[arenaID]
		arenaOffset = 0
		arenaRest = min(rest, ArenaSize)
		if rest > 0 {
			goto loop
		}
	}

	s.m().Hit()
	return dst, ErrOK
}

func (s *bucket) bulkEvict() error {
	if err := s.checkStatus(); err != nil {
		return err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	el := s.elen()
	if el == 0 {
		return ErrOK
	}

	entry := s.entry
	now := s.now()
	_ = entry[el-1]
	z := sort.Search(int(el), func(i int) bool {
		return now <= entry[i].expire
	})

	if z == 0 {
		return ErrOK
	}

	arenaID := s.entry[z].arenaID()

	var wg sync.WaitGroup

	wg.Add(1)
	go s.evictRange(&wg, z)

	wg.Add(1)
	go s.recycleArena(&wg, arenaID)

	wg.Wait()

	return ErrOK
}

func (s *bucket) recycleArena(wg *sync.WaitGroup, arenaID uint32) {
	defer wg.Done()
	var arenaIdx int
	al := len(s.arena)
	if al == 0 {
		return
	}
	if al < 256 {
		_ = s.arena[al-1]
		for i := 0; i < al; i++ {
			if s.arena[i].id == arenaID {
				arenaIdx = i
				break
			}
		}
	} else {
		al8 := al - al%8
		_ = s.arena[al-1]
		for i := 0; i < al8; i += 8 {
			if s.arena[i].id == arenaID {
				arenaIdx = i
				break
			}
			if s.arena[i+1].id == arenaID {
				arenaIdx = i + 1
				break
			}
			if s.arena[i+2].id == arenaID {
				arenaIdx = i + 2
				break
			}
			if s.arena[i+3].id == arenaID {
				arenaIdx = i + 3
				break
			}
			if s.arena[i+4].id == arenaID {
				arenaIdx = i + 4
				break
			}
			if s.arena[i+5].id == arenaID {
				arenaIdx = i + 5
				break
			}
			if s.arena[i+6].id == arenaID {
				arenaIdx = i + 6
				break
			}
			if s.arena[i+7].id == arenaID {
				arenaIdx = i + 7
				break
			}
		}
	}
	if arenaIdx == 0 {
		return
	}

	s.arenaBuf = append(s.arenaBuf[:0], s.arena[:arenaIdx]...)
	copy(s.arena, s.arena[arenaIdx:])
	s.arena = append(s.arena[:arenaIdx], s.arenaBuf...)
	s.m().Free(uint32(len(s.arenaBuf)) * ArenaSize)

	_ = s.arena[al-1]
	for i := 0; i < al; i++ {
		s.arena[i].id = uint32(i)
	}
}

func (s *bucket) evictRange(wg *sync.WaitGroup, z int) {
	defer wg.Done()
	el := s.elen()
	if z < 256 {
		_ = s.entry[el-1]
		for i := 0; i < z; i++ {
			s.evict(&s.entry[i])
		}
	} else {
		z8 := z - z%8
		_ = s.entry[el-1]
		for i := 0; i < z8; i += 8 {
			s.evict(&s.entry[i])
			s.evict(&s.entry[i+1])
			s.evict(&s.entry[i+2])
			s.evict(&s.entry[i+3])
			s.evict(&s.entry[i+4])
			s.evict(&s.entry[i+5])
			s.evict(&s.entry[i+6])
			s.evict(&s.entry[i+7])
		}
		for i := z8; i < z; i++ {
			s.evict(&s.entry[i])
		}
	}

	copy(s.entry, s.entry[z:])
	s.entry = s.entry[:el-uint32(z)]
}

func (s *bucket) evict(e *entry) {
	s.m().Evict(e.length)
	delete(s.index, e.hash)
}

func (s *bucket) checkStatus() error {
	if status := atomic.LoadUint32(&s.status); status != bucketStatusActive {
		if status == bucketStatusService {
			return ErrBucketService
		}
		if status == bucketStatusCorrupt {
			// todo return corresponding error.
		}
	}
	return nil
}

func (s *bucket) now() uint32 {
	return atomic.LoadUint32(s.nowPtr)
}

func (s *bucket) alen() uint32 {
	return uint32(len(s.arena))
}

func (s *bucket) elen() uint32 {
	return uint32(len(s.entry))
}

func (s *bucket) m() MetricsWriter {
	return s.config.MetricsWriter
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
