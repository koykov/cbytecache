package cbytecache

import (
	"encoding/binary"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/koykov/cbytebuf"
	"github.com/koykov/fastconv"
)

type bucket struct {
	config  *Config
	idx     uint32
	status  uint32
	maxSize uint32
	actSize uint32

	mux   sync.RWMutex
	buf   *cbytebuf.CByteBuf
	index map[uint64]uint32
	entry []entry

	arena, arenaBuf []arena

	arenaOffset uint32
}

func (b *bucket) set(key string, h uint64, p []byte) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	err = b.setLF(key, h, p)
	b.mux.Unlock()
	return
}

func (b *bucket) setm(key string, h uint64, m MarshallerTo) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	defer b.mux.Unlock()
	b.buf.ResetLen()
	if _, err = b.buf.WriteMarshallerTo(m); err != nil {
		return
	}
	err = b.setLF(key, h, b.buf.Bytes())
	return
}

func (b *bucket) setLF(key string, h uint64, p []byte) (err error) {
	var (
		idx, pl uint32

		e  *entry
		ok bool
	)

	defer b.buf.ResetLen()

	if idx, ok = b.index[h]; ok {
		if idx < b.elen() {
			e = &b.entry[idx]
		}
	}

	if p, pl, err = b.c7n(key, p); err != nil {
		return
	}

	if e != nil {
		if b.config.CollisionCheck {
			bl := b.buf.Len()
			if err = b.buf.GrowDelta(int(e.length)); err != nil {
				return
			}
			dst := b.buf.Bytes()[bl:]
			if dst, err = b.getLF(dst[:0], e, dummyMetrics); err != nil {
				return
			}
			if len(dst) < keySizeBytes {
				err = ErrEntryCorrupt
				return
			}
			kl := binary.LittleEndian.Uint16(dst[:keySizeBytes])
			dst = dst[keySizeBytes:]
			if kl >= uint16(len(dst)) {
				err = ErrEntryCorrupt
				return
			}
			key1 := fastconv.B2S(dst[:kl])
			if key1 != key {
				b.l().Printf("collision %d: keys \"%s\" and \"%s\"", h, key, key1)
				b.m().Collision(b.k())
				err = ErrEntryCollision
				return
			}
		}
		err = ErrEntryExists
		return
	}

	if b.arenaOffset >= b.alen() {
		if b.maxSize > 0 && b.alen()*ArenaSize+ArenaSize > b.maxSize {
			b.l().Printf("bucket %d: no space on grow", b.idx)
			b.m().NoSpace(b.k())
			return ErrNoSpace
		}
		for {
			b.m().Alloc(b.k(), ArenaSize)
			arena := allocArena(b.alen())
			b.arena = append(b.arena, *arena)
			if b.alen() > b.arenaOffset {
				break
			}
		}
	}
	arena := &b.arena[b.arenaOffset]
	arenaID := uintptr(unsafe.Pointer(&arena.id))
	arenaOffset, arenaRest := arena.offset(), arena.rest()
	rest := uint32(len(p))
	if arenaRest >= rest {
		arena.write(p)
	} else {
		mustWrite := arenaRest
		for {
			arena.write(p[:mustWrite])
			p = p[mustWrite:]
			if rest -= mustWrite; rest == 0 {
				break
			}
			b.arenaOffset++
			if b.arenaOffset >= b.alen() {
				if b.maxSize > 0 && b.alen()*ArenaSize+ArenaSize > b.maxSize {
					b.l().Printf("bucket %d: no space on write", b.idx)
					b.m().NoSpace(b.k())
					return ErrNoSpace
				}
				for {
					b.m().Alloc(b.k(), ArenaSize)
					arena := allocArena(b.alen())
					b.arena = append(b.arena, *arena)
					if b.alen() > b.arenaOffset {
						break
					}
				}
			}
			arena = &b.arena[b.arenaOffset]
			mustWrite = min(rest, ArenaSize)
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

	atomic.AddUint32(&b.actSize, pl)
	b.m().Set(b.k(), pl)
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
		b.m().Miss(b.k())
		return dst, ErrNotFound
	}
	if idx >= b.elen() {
		b.m().Miss(b.k())
		return dst, ErrNotFound
	}
	entry := &b.entry[idx]
	if entry.expire < b.now() {
		b.m().Expire(b.k())
		return dst, ErrNotFound
	}

	return b.getLF(dst, entry, b.m())
}

func (b *bucket) getLF(dst []byte, entry *entry, mw MetricsWriter) ([]byte, error) {
	arenaID := entry.arenaID()
	arenaOffset := entry.offset

	if arenaID >= b.alen() {
		mw.Miss(b.k())
		return dst, ErrNotFound
	}
	arena := &b.arena[arenaID]

	arenaRest := ArenaSize - arenaOffset
	if entry.offset+entry.length < ArenaSize {
		dst = append(dst, arena.read(arenaOffset, entry.length)...)
	} else {
		rest := entry.length
	loop:
		dst = append(dst, arena.read(arenaOffset, arenaRest)...)
		rest -= arenaRest
		arenaID++
		if arenaID >= b.alen() {
			mw.Corrupt(b.k())
			return dst, ErrEntryCorrupt
		}
		arena = &b.arena[arenaID]
		arenaOffset = 0
		arenaRest = min(rest, ArenaSize)
		if rest > 0 {
			goto loop
		}
	}
	mw.Hit(b.k())
	return dst, ErrOK
}

func (b *bucket) c7n(key string, p []byte) ([]byte, uint32, error) {
	pl := uint32(len(p))
	if !b.config.CollisionCheck {
		return p, pl, nil
	}
	var err error
	bl := b.buf.Len()
	if err = b.buf.GrowLen(bl + keySizeBytes); err != nil {
		return p, pl, err
	}
	kl := uint16(len(key))
	binary.LittleEndian.PutUint16(b.buf.Bytes()[bl:], kl)
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

func (b *bucket) size() uint32 {
	return atomic.LoadUint32(&b.actSize)
}

func (b *bucket) bulkEvict() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	b.l().Printf("bucket #%d: bulk evict started", b.idx)

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		b.l().Printf("bucket #%d: bulk evict finished", b.idx)
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
	}()

	el := b.elen()
	if el == 0 {
		return ErrOK
	}

	entry := b.entry
	now := b.now()
	_ = entry[el-1]
	z := sort.Search(int(el), func(i int) bool {
		return now <= entry[i].expire
	})

	if z == 0 {
		return ErrOK
	}

	arenaID := b.entry[z-1].arenaID()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		b.evictRange(z)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		bal, bao := b.alen(), b.arenaOffset
		b.recycleArena(arenaID)
		aal, aao := b.alen(), b.arenaOffset
		b.l().Printf("bucket #%d: arena len/offset %d/%d -> %d/%d", b.idx, bal, bao, aal, aao)
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
		for i := al8; i < al; i++ {
			if b.arena[i].id == arenaID {
				arenaIdx = i
				break
			}
		}
	}
	if arenaIdx == 0 {
		return
	}

	b.arenaBuf = append(b.arenaBuf[:0], b.arena[:arenaIdx]...)
	copy(b.arena, b.arena[arenaIdx:])
	b.arenaOffset = uint32(al - arenaIdx - 1)
	b.arena = append(b.arena[:b.arenaOffset+1], b.arenaBuf...)

	_ = b.arena[al-1]
	for i := 0; i < al; i++ {
		b.arena[i].id = uint32(i)
		if i >= arenaIdx {
			b.arena[i].h.Len = 0
		}
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
	atomic.AddUint32(&b.actSize, math.MaxUint32-e.length+1)
	b.m().Evict(b.k(), e.length)
	delete(b.index, e.hash)
}

func (b *bucket) vacuum() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	b.l().Printf("bucket #%d: vacuum started", b.idx)

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		b.l().Printf("bucket #%d: vacuum finished", b.idx)
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
	}()

	ad := b.alen() - b.arenaOffset
	if ad <= 1 {
		return ErrOK
	}
	pos := b.alen() - ad + 1
	for i := pos; i < b.alen(); i++ {
		b.arena[i].release()
	}
	b.arena = b.arena[:pos]
	b.arenaBuf = b.arenaBuf[:0]

	return ErrOK
}

func (b *bucket) reset() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
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

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
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
	return ErrOK
}

func (b *bucket) now() uint32 {
	return uint32(b.config.Clock.Now().Unix())
}

func (b *bucket) alen() uint32 {
	return uint32(len(b.arena))
}

func (b *bucket) elen() uint32 {
	return uint32(len(b.entry))
}

func (b *bucket) k() string {
	return b.config.Key
}

func (b *bucket) m() MetricsWriter {
	return b.config.MetricsWriter
}

func (b *bucket) l() Logger {
	return b.config.Logger
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
