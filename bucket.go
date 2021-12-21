package cbytecache

import (
	"encoding/binary"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/koykov/cbytebuf"
	"github.com/koykov/fastconv"
)

// Cache bucket.
type bucket struct {
	config  *Config
	idx     uint32
	status  uint32
	sizeMax uint32
	size    bucketSize

	mux sync.RWMutex
	// Internal buffer.
	buf *cbytebuf.CByteBuf
	// Entry index. Value point to the index in entry array.
	index map[uint64]uint32
	// Entries storage.
	entry []entry

	// Memory arenas.
	arena, arenaBuf []arena

	// Index of current arena available to write data.
	arenaOffset uint32
}

// Set p to bucket by h hash.
func (b *bucket) set(key string, h uint64, p []byte) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	err = b.setLF(key, h, p)
	b.mux.Unlock()
	return
}

// Set m to bucket by h hash.
func (b *bucket) setm(key string, h uint64, m MarshallerTo) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	defer b.mux.Unlock()
	// Use internal buffer to convert m to bytes before set.
	b.buf.ResetLen()
	if _, err = b.buf.WriteMarshallerTo(m); err != nil {
		return
	}
	err = b.setLF(key, h, b.buf.Bytes())
	return
}

// Internal setter. It works in lock-free mode thus need to guarantee thread-safety outside.
func (b *bucket) setLF(key string, h uint64, p []byte) (err error) {
	var (
		idx, pl uint32

		e  *entry
		ok bool
	)

	defer b.buf.ResetLen()

	// Try to get already existed entry.
	if idx, ok = b.index[h]; ok {
		if idx < b.elen() {
			e = &b.entry[idx]
		}
	}

	// Extend entry data with collision control data.
	if p, pl, err = b.c7n(key, p); err != nil {
		return
	}

	// Cache doesn't support of overwrite existing entries, but it's handy to use known existing entry for collision check.
	if e != nil {
		if b.config.CollisionCheck {
			// Prepare space in buffer to get potentially collided entry.
			bl := b.buf.Len()
			if err = b.buf.GrowDelta(int(e.length)); err != nil {
				return
			}
			dst := b.buf.Bytes()[bl:]
			// Read entry bytes.
			if dst, err = b.getLF(dst[:0], e, dummyMetrics); err != nil {
				return
			}
			if len(dst) < keySizeBytes {
				err = ErrEntryCorrupt
				return
			}
			// Extract raw key from entry body.
			kl := binary.LittleEndian.Uint16(dst[:keySizeBytes])
			dst = dst[keySizeBytes:]
			if kl >= uint16(len(dst)) {
				err = ErrEntryCorrupt
				return
			}
			key1 := fastconv.B2S(dst[:kl])
			if key1 != key {
				// Keys don't match - collision caught.
				if b.l() != nil {
					b.l().Printf("collision %d: keys \"%s\" and \"%s\"", h, key, key1)
				}
				b.m().Collision(b.k())
				err = ErrEntryCollision
				return
			}
		}
		// Exit anyway.
		err = ErrEntryExists
		return
	}

	// Check if more space needed before write.
	if b.arenaOffset >= b.alen() {
		if b.sizeMax > 0 && b.alen()*ArenaSize+ArenaSize > b.sizeMax {
			if b.l() != nil {
				b.l().Printf("bucket %d: no space on grow", b.idx)
			}
			b.m().NoSpace(b.k())
			return ErrNoSpace
		}
		for {
			b.m().Alloc(b.k(), ArenaSize)
			arena := allocArena(b.alen())
			b.size.snap(snapAlloc, ArenaSize)
			b.arena = append(b.arena, *arena)
			if b.alen() > b.arenaOffset {
				break
			}
		}
	}
	// Get current arena.
	arena := &b.arena[b.arenaOffset]
	arenaID := uintptr(unsafe.Pointer(&arena.id))
	arenaOffset, arenaRest := arena.offset(), arena.rest()
	rest := uint32(len(p))
	if arenaRest >= rest {
		// Arena has enough space to write the entry.
		arena.write(p)
	} else {
		// Arena hasn't enough space - need share entry among arenas.
		mustWrite := arenaRest
		for {
			// Write entry bytes that fits to arena free space.
			arena.write(p[:mustWrite])
			p = p[mustWrite:]
			if rest -= mustWrite; rest == 0 {
				// All entry bytes written.
				break
			}
			// Switch to the next arena.
			b.arenaOffset++
			// Alloc new arena if needed.
			if b.arenaOffset >= b.alen() {
				if b.sizeMax > 0 && b.alen()*ArenaSize+ArenaSize > b.sizeMax {
					if b.l() != nil {
						b.l().Printf("bucket %d: no space on write", b.idx)
					}
					b.m().NoSpace(b.k())
					return ErrNoSpace
				}
				for {
					b.m().Alloc(b.k(), ArenaSize)
					arena := allocArena(b.alen())
					b.size.snap(snapAlloc, ArenaSize)
					b.arena = append(b.arena, *arena)
					if b.alen() > b.arenaOffset {
						break
					}
				}
			}
			arena = &b.arena[b.arenaOffset]
			// Calculate rest of bytes to write.
			mustWrite = min(rest, ArenaSize)
		}
	}

	// Create and register new entry.
	b.entry = append(b.entry, entry{
		hash:   h,
		offset: arenaOffset,
		length: pl,
		expire: b.now() + uint32(b.config.Expire)/1e9,
		aidptr: arenaID,
	})
	b.index[h] = b.elen() - 1

	b.size.snap(snapSet, pl)
	b.m().Set(b.k(), pl)
	return ErrOK
}

// Get entry by h hash.
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

// Internal getter. It works in lock-free mode thus need to guarantee thread-safety outside.
func (b *bucket) getLF(dst []byte, entry *entry, mw MetricsWriter) ([]byte, error) {
	// Get starting arena.
	arenaID := entry.arenaID()
	arenaOffset := entry.offset

	if arenaID >= b.alen() {
		mw.Miss(b.k())
		return dst, ErrNotFound
	}
	arena := &b.arena[arenaID]

	arenaRest := ArenaSize - arenaOffset
	if entry.offset+entry.length < ArenaSize {
		// Good case: entry doesn't share among arenas.
		dst = append(dst, arena.read(arenaOffset, entry.length)...)
	} else {
		// Walk over arenas and collect entry by parts.
		rest := entry.length
		for rest > 0 {
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
		}
	}
	mw.Hit(b.k())
	return dst, ErrOK
}

// Extend entry with collision control data.
func (b *bucket) c7n(key string, p []byte) ([]byte, uint32, error) {
	pl := uint32(len(p))
	if !b.config.CollisionCheck {
		return p, pl, nil
	}
	var err error
	// Write key length to the first two bytes.
	bl := b.buf.Len()
	if err = b.buf.GrowLen(bl + keySizeBytes); err != nil {
		return p, pl, err
	}
	kl := uint16(len(key))
	binary.LittleEndian.PutUint16(b.buf.Bytes()[bl:], kl)
	// Write key ...
	if _, err = b.buf.WriteString(key); err != nil {
		return p, pl, err
	}
	// ... and body.
	if _, err = b.buf.Write(p); err != nil {
		return p, pl, err
	}
	// p extends now with collision control data.
	p = b.buf.Bytes()[bl:]
	pl = uint32(len(p))
	return p, pl, err
}

func (b *bucket) bulkEvict() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	if b.l() != nil {
		b.l().Printf("bucket #%d: bulk evict started", b.idx)
	}

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: bulk evict finished", b.idx)
		}
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
		if b.l() != nil {
			b.l().Printf("bucket #%d: arena len/offset %d/%d -> %d/%d", b.idx, bal, bao, aal, aao)
		}
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
	b.size.snap(snapEvict, e.length)
	b.m().Evict(b.k(), e.length)
	delete(b.index, e.hash)
}

func (b *bucket) vacuum() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	if b.l() != nil {
		b.l().Printf("bucket #%d: vacuum started", b.idx)
	}

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: vacuum finished", b.idx)
		}
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
		b.size.snap(snapRelease, ArenaSize)
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
			b.size.snap(snapRelease, ArenaSize)
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
