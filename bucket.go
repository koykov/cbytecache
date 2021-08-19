package cbytecache

import (
	"encoding/binary"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/koykov/cbytebuf"
	"github.com/koykov/fastconv"
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

func (b *bucket) set(key string, h uint64, p []byte, force bool) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	err = b.setLF(key, h, p, force)
	b.mux.Unlock()
	return
}

func (b *bucket) setm(key string, h uint64, m MarshallerTo, force bool) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	defer b.mux.Unlock()
	b.buf.ResetLen()
	if _, err = b.buf.WriteMarshallerTo(m); err != nil {
		return
	}
	err = b.setLF(key, h, b.buf.Bytes(), force)
	return
}

func (b *bucket) setLF(key string, h uint64, p []byte, force bool) error {
	// Look for existing entry to reset or update it.
	var (
		e    *entry
		pl   uint32
		idx  uint32
		idxl bool
		ok   bool
		err  error
	)
	if idx, ok = b.index[h]; ok {
		if idx < b.elen() {
			e = &b.entry[idx]
			idxl = idx == b.elen()-1
		}
	}

	if p, pl, err = b.c7n(key, p); err != nil {
		return err
	}

	if e != nil {
		if b.config.CollisionCheck {
			bl := b.buf.Len()
			if err = b.buf.GrowDelta(int(e.length)); err != nil {
				return err
			}
			dst := b.buf.Bytes()[bl:]
			if dst, err = b.getLF(dst[:0], e, &DummyMetrics{}); err != nil {
				return err
			}
			if len(dst) < keySizeBytes {
				return ErrEntryCorrupt
			}
			kl := binary.LittleEndian.Uint32(dst[:keySizeBytes])
			dst = dst[4:]
			if kl >= uint32(len(dst)) {
				return ErrEntryCorrupt
			}
			key1 := fastconv.B2S(dst[:kl])
			if key1 != key {
				b.m().Collision()
				return ErrEntryCollision
			}
		}
		if e.expire >= b.now() && !force && !idxl {
			return ErrEntryExists
		}
		e.hash = 0
	}

	if idxl {
		// todo update existing entry at the end of the bucket instead of create new one.
	}

	if b.arenaOffset >= b.alen() {
		if b.maxSize > 0 && b.alen()*ArenaSize+ArenaSize > b.maxSize {
			b.m().NoSpace()
			return ErrNoSpace
		}
	alloc1:
		b.m().Alloc(ArenaSize)
		arena := allocArena(b.alen())
		b.arena = append(b.arena, *arena)
		if b.alen() <= b.arenaOffset {
			goto alloc1
		}
	}
	arena := &b.arena[b.arenaOffset]
	arenaID := &arena.id
	arenaOffset := arena.offset
	arenaRest := ArenaSize - arena.offset
	rest := uint32(len(p))
	if arenaRest >= rest {
		arena.bytesCopy(arena.offset, p)
		arena.offset += rest
	} else {
	loop:
		arena.bytesCopy(arena.offset, p[:arenaRest])
		p = p[arenaRest:]
		rest -= arenaRest
		b.arenaOffset++
		if b.arenaOffset >= b.alen() {
			if b.maxSize > 0 && b.alen()*ArenaSize+ArenaSize > b.maxSize {
				return ErrNoSpace
			}
		alloc2:
			b.m().Alloc(ArenaSize)
			arena := allocArena(b.alen())
			b.arena = append(b.arena, *arena)
			if b.alen() <= b.arenaOffset {
				goto alloc2
			}
		}
		arena = &b.arena[b.arenaOffset]
		arenaRest = min(rest, ArenaSize)
		if rest > 0 {
			goto loop
		}
	}

	b.entry = append(b.entry, entry{
		hash:   h,
		offset: arenaOffset,
		length: pl,
		expire: b.now() + uint32(b.config.Expire)/1e9,
		aidptr: arenaID,
	})
	b.index[h] = b.elen() - 1

	b.m().Set(pl)
	return ErrOK
}

func (b *bucket) get(dst []byte, h uint64) ([]byte, error) {
	if err := b.checkStatus(); err != nil {
		return dst, err
	}

	b.mux.RLock()
	defer b.mux.RUnlock()
	var (
		idx uint32
		ok  bool
	)
	if idx, ok = b.index[h]; !ok {
		b.m().Miss()
		return dst, ErrNotFound
	}
	if idx >= b.elen() {
		b.m().Miss()
		return dst, ErrNotFound
	}
	entry := &b.entry[idx]
	if entry.expire < b.now() {
		b.m().Expire()
		return dst, ErrNotFound
	}

	return b.getLF(dst, entry, b.m())
}

func (b *bucket) getLF(dst []byte, entry *entry, mw MetricsWriter) ([]byte, error) {
	arenaID := entry.arenaID()
	arenaOffset := entry.offset

	if arenaID >= b.alen() {
		mw.Miss()
		return dst, ErrNotFound
	}
	arena := &b.arena[arenaID]

	arenaRest := ArenaSize - arenaOffset
	if entry.offset+entry.length < ArenaSize {
		dst = append(dst, arena.bytesRange(arenaOffset, entry.length)...)
	} else {
		rest := entry.length
	loop:
		dst = append(dst, arena.bytesRange(arenaOffset, arenaRest)...)
		rest -= arenaRest
		arenaID++
		if arenaID >= b.alen() {
			mw.Corrupt()
			return dst, ErrEntryCorrupt
		}
		arena = &b.arena[arenaID]
		arenaOffset = 0
		arenaRest = min(rest, ArenaSize)
		if rest > 0 {
			goto loop
		}
	}
	mw.Hit()
	return dst, ErrOK
}

func (b *bucket) c7n(key string, p []byte) ([]byte, uint32, error) {
	pl := uint32(len(p))
	if !b.config.CollisionCheck {
		return p, pl, nil
	}
	var err error
	bl := b.buf.Len()
	if err = b.buf.GrowDelta(keySizeBytes); err != nil {
		return p, pl, err
	}
	kl := uint16(len(key))
	binary.LittleEndian.PutUint16(b.buf.Bytes(), kl)
	if _, err = b.buf.WriteString(key); err != nil {
		return p, pl, err
	}
	if _, err = b.buf.Write(p); err != nil {
		return p, pl, err
	}
	p = b.buf.Bytes()[bl:]
	pl = uint32(len(p))
	return p, pl, err
}

func (b *bucket) bulkEvict() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	b.mux.Lock()
	b.status = bucketStatusService
	defer func() {
		b.status = bucketStatusActive
		b.mux.Unlock()
	}()

	el := b.elen()
	if el == 0 {
		return ErrOK
	}

	entry := b.entry
	now := b.now()
	_ = entry[el-1]
	z := sort.Search(int(el-1), func(i int) bool {
		return now <= entry[i].expire
	})

	if z == 0 {
		return ErrOK
	}

	arenaID := b.entry[z].arenaID()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		b.evictRange(z)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		b.recycleArena(arenaID)
		wg.Done()
	}()

	wg.Wait()

	return ErrOK
}

func (b *bucket) recycleArena(arenaID uint32) {
	var arenaIdx int
	al := len(b.arena)
	if al == 0 {
		return
	}
	if al < 256 {
		_ = b.arena[al-1]
		for i := 0; i < al; i++ {
			if b.arena[i].id == arenaID {
				arenaIdx = i
				break
			}
		}
	} else {
		al8 := al - al%8
		_ = b.arena[al-1]
		for i := 0; i < al8; i += 8 {
			if b.arena[i].id == arenaID {
				arenaIdx = i
				break
			}
			if b.arena[i+1].id == arenaID {
				arenaIdx = i + 1
				break
			}
			if b.arena[i+2].id == arenaID {
				arenaIdx = i + 2
				break
			}
			if b.arena[i+3].id == arenaID {
				arenaIdx = i + 3
				break
			}
			if b.arena[i+4].id == arenaID {
				arenaIdx = i + 4
				break
			}
			if b.arena[i+5].id == arenaID {
				arenaIdx = i + 5
				break
			}
			if b.arena[i+6].id == arenaID {
				arenaIdx = i + 6
				break
			}
			if b.arena[i+7].id == arenaID {
				arenaIdx = i + 7
				break
			}
		}
	}
	if arenaIdx == 0 {
		return
	}

	b.arenaBuf = append(b.arenaBuf[:0], b.arena[:arenaIdx]...)
	copy(b.arena, b.arena[arenaIdx:])
	b.arena = append(b.arena[:arenaIdx], b.arenaBuf...)
	b.m().Free(uint32(len(b.arenaBuf)) * ArenaSize)

	_ = b.arena[al-1]
	for i := 0; i < al; i++ {
		b.arena[i].id = uint32(i)
	}
}

func (b *bucket) evictRange(z int) {
	el := b.elen()
	if z < 256 {
		_ = b.entry[el-1]
		for i := 0; i < z; i++ {
			b.evict(&b.entry[i])
		}
	} else {
		z8 := z - z%8
		_ = b.entry[el-1]
		for i := 0; i < z8; i += 8 {
			b.evict(&b.entry[i])
			b.evict(&b.entry[i+1])
			b.evict(&b.entry[i+2])
			b.evict(&b.entry[i+3])
			b.evict(&b.entry[i+4])
			b.evict(&b.entry[i+5])
			b.evict(&b.entry[i+6])
			b.evict(&b.entry[i+7])
		}
		for i := z8; i < z; i++ {
			b.evict(&b.entry[i])
		}
	}

	copy(b.entry, b.entry[z:])
	b.entry = b.entry[:el-uint32(z)]
}

func (b *bucket) evict(e *entry) {
	b.m().Evict(e.length)
	delete(b.index, e.hash)
}

func (b *bucket) reset() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	b.mux.Lock()
	b.status = bucketStatusService
	defer func() {
		b.status = bucketStatusActive
		b.mux.Unlock()
	}()

	b.buf.ResetLen()
	b.arenaOffset = 0
	b.evictRange(len(b.entry))
	b.entry = b.entry[:0]

	return ErrOK
}

func (b *bucket) release() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	b.mux.Lock()
	b.status = bucketStatusService
	defer func() {
		b.status = bucketStatusActive
		b.mux.Unlock()
	}()

	b.buf.Release()
	b.arenaOffset = 0

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if b.elen() == 0 {
			return
		}
		b.evictRange(len(b.entry))
		b.entry = b.entry[:0]
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		al := b.alen()
		if al == 0 {
			return
		}
		_ = b.arena[al-1]
		var i uint32
		for i = 0; i < al; i++ {
			b.arena[i].release()
		}
	}()

	return ErrOK
}

func (b *bucket) checkStatus() error {
	if status := atomic.LoadUint32(&b.status); status != bucketStatusActive {
		if status == bucketStatusService {
			return ErrBucketService
		}
		if status == bucketStatusCorrupt {
			return ErrBucketCorrupt
		}
	}
	return nil
}

func (b *bucket) now() uint32 {
	return atomic.LoadUint32(b.nowPtr)
}

func (b *bucket) alen() uint32 {
	return uint32(len(b.arena))
}

func (b *bucket) elen() uint32 {
	return uint32(len(b.entry))
}

func (b *bucket) m() MetricsWriter {
	return b.config.MetricsWriter
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
