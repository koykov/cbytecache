package cbytecache

import (
	"encoding/binary"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/koykov/cbytebuf"
	"github.com/koykov/fastconv"
)

// Cache bucket.
type bucket struct {
	config *Config
	idx    uint32
	ids    string
	status uint32
	maxCap uint32
	size   bucketSize

	mux sync.RWMutex
	// Internal buffer.
	buf *cbytebuf.CByteBuf
	// Entry index. Value point to the index in entry array.
	index map[uint64]uint32
	// Entries storage.
	entry []entry

	// Memory arenas.
	arena arenaList

	lastEvc, lastVac time.Time

	// Index of current arena available to write data.
	// todo remove me
	arenaIdx uint32
}

// Set p to bucket by h hash.
func (b *bucket) set(key string, h uint64, p []byte) (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	b.mux.Lock()
	err = b.setLF(key, h, p, 0)
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
	err = b.setLF(key, h, b.buf.Bytes(), 0)
	return
}

// Internal setter. It works in lock-free mode thus need to guarantee thread-safety outside.
func (b *bucket) setLF(key string, h uint64, p []byte, expire uint32) (err error) {
	var (
		idx, pl uint32

		e   *entry
		ok  bool
		stm = b.nowT()
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
			if err = b.buf.GrowLen(bl + int(e.length)); err != nil {
				return
			}
			p = b.buf.Bytes()[:bl] // update p due to possible buffer grow
			dst := b.buf.Bytes()[bl:]
			// Read entry key and payload.
			var key1 string
			if key1, dst, err = b.getLF(dst[:0], e, dummyMetrics); err != nil {
				return
			}
			if key1 != key {
				// Keys don't match - collision caught.
				if b.l() != nil {
					b.l().Printf("collision %d: keys \"%s\" and \"%s\"", h, key, key1)
				}
				b.mw().Collision(b.ids)
				err = ErrEntryCollision
				return
			}
		}
		// Exit anyway.
		err = ErrEntryExists
		return
	}

	// Init alloc.
	if b.arena.len() == 0 {
		a := b.arena.alloc(nil, b.as())
		b.arena.setHead(a).setAct(a)
		b.mw().Alloc(b.ids)
		b.size.snap(snapAlloc, b.ac32())
	}
	// Get current arena.
	a := b.arena.act()
	startArena := a
	arenaOffset, arenaRest := a.offset(), a.rest()
	rest := uint32(len(p))
	if arenaRest >= rest {
		// Arena has enough space to write the entry.
		a.write(p)
	} else {
		// Arena hasn't enough space - need share entry among arenas.
		mustWrite := arenaRest
		for {
			// Write entry bytes that fits to arena free space.
			a.write(p[:mustWrite])
			p = p[mustWrite:]
			if rest -= mustWrite; rest == 0 {
				// All entry bytes written.
				break
			}
			// Switch to the next arena.
			prev := a
			a = a.next()
			b.arena.setAct(a)
			b.mw().Fill(b.ids)
			// Alloc new arena if needed.
			if a == nil {
				if b.maxCap > 0 && b.alen()*b.ac32()+b.ac32() > b.maxCap {
					if b.l() != nil {
						b.l().Printf("bucket %d: no space on write", b.idx)
					}
					b.mw().NoSpace(b.ids)
					return ErrNoSpace
				}
				a = b.arena.alloc(prev, b.as())
				b.mw().Alloc(b.ids)
				prev.setNext(a)
				b.arena.setAct(a).setTail(a)
				b.size.snap(snapAlloc, b.ac32())
			}
			// Calculate rest of bytes to write.
			mustWrite = min(rest, b.ac32())
		}
	}

	// Create and register new entry.
	entry := entry{
		hash:   h,
		offset: arenaOffset,
		length: pl,
		expire: expire,
		aptr:   startArena.ptr(),
	}
	if entry.expire == 0 {
		entry.expire = uint32(b.config.Clock.Now().Add(b.config.ExpireInterval).Unix())
	}
	b.entry = append(b.entry, entry)
	b.index[h] = b.elen() - 1

	b.size.snap(snapSet, pl)
	b.mw().Set(b.ids, b.nowT().Sub(stm))
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
		stm = b.nowT()
	)
	if idx, ok = b.index[h]; !ok {
		b.mw().Miss(b.ids)
		return dst, ErrNotFound
	}
	if idx >= b.elen() {
		b.mw().Miss(b.ids)
		return dst, ErrNotFound
	}
	entry := &b.entry[idx]
	if entry.expire < b.now() {
		b.mw().Expire(b.ids)
		return dst, ErrNotFound
	}

	var err error
	_, dst, err = b.getLF(dst, entry, b.mw())
	if err == nil {
		b.mw().Hit(b.ids, b.nowT().Sub(stm))
	}
	return dst, err
}

// Internal getter. It works in lock-free mode thus need to guarantee thread-safety outside.
func (b *bucket) getLF(dst []byte, entry *entry, mw MetricsWriter) (string, []byte, error) {
	// Get starting arena.
	arenaOffset := entry.offset

	arena := entry.arena()
	if arena == nil {
		mw.Miss(b.ids)
		return "", dst, ErrNotFound
	}

	arenaRest := b.ac32() - arenaOffset
	if entry.offset+entry.length < b.ac32() {
		// Good case: entry doesn't share among arenas.
		dst = append(dst, arena.read(arenaOffset, entry.length)...)
	} else {
		// Walk over arenas and collect entry by parts.
		rest := entry.length
		for rest > 0 {
			dst = append(dst, arena.read(arenaOffset, arenaRest)...)
			if rest -= arenaRest; rest == 0 {
				break
			}
			arena = arena.next()
			if arena == nil {
				mw.Corrupt(b.ids)
				return "", dst, ErrEntryCorrupt
			}
			arenaOffset = 0
			arenaRest = min(rest, b.ac32())
		}
	}

	if len(dst) < keySizeBytes {
		return "", dst, ErrEntryCorrupt
	}
	l := len(dst)
	kl := binary.LittleEndian.Uint16(dst[l-keySizeBytes:])
	if l-2 <= int(kl) {
		return "", dst, ErrEntryCorrupt
	}
	key := fastconv.B2S(dst[l-int(kl)-keySizeBytes : l-keySizeBytes])

	return key, dst[:l-int(kl)-keySizeBytes], ErrOK
}

// Extend entry with collision control data.
func (b *bucket) c7n(key string, p []byte) ([]byte, uint32, error) {
	pl := uint32(len(p))
	var err error
	// Write payload at start.
	if _, err = b.buf.Write(p); err != nil {
		return p, pl, err
	}
	// Write key.
	if _, err = b.buf.WriteString(key); err != nil {
		return p, pl, err
	}
	// Write key length to the last two bytes.
	bl := b.buf.Len()
	if err = b.buf.GrowLen(bl + keySizeBytes); err != nil {
		return p, pl, err
	}
	kl := uint16(len(key))
	binary.LittleEndian.PutUint16(b.buf.Bytes()[bl:], kl)
	// p extends now with collision control data.
	p = b.buf.Bytes()
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

	return b.bulkEvictLF(false)
}

func (b *bucket) bulkEvictLF(force bool) error {
	if !force && b.nowT().Sub(b.lastEvc) < b.config.EvictInterval/10*9 {
		return ErrOK
	}

	defer func() {
		b.lastEvc = b.nowT()
		b.flushMetricsLF()
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
	if z == 0 || z == int(el) {
		return ErrOK
	}

	// Last arena contains unexpired entries.
	lo := b.entry[z-1].arena()
	// Previous arena must contain only expired entries.
	lo1 := lo.prev()

	if b.config.ExpireListener != nil {
		// Call expire listener for all expired entries.
		b.expireRange(z)
	}

	// Async evict/recycle.
	var wg sync.WaitGroup

	// Evict all expired entries.
	wg.Add(1)
	go func() {
		b.evictRange(z)
		if b.l() != nil {
			b.l().Printf("bucket #%d: evict entry %d", b.idx, z)
		}
		wg.Done()
	}()

	// Recycle arenas.
	wg.Add(1)
	go func() {
		b.arena.recycle(lo1)

		var c int
		a := b.arena.act().next()
		for a != nil {
			if !a.empty() {
				a.reset()
				b.mw().Reset(b.ids)
				c++
			}
			a = a.next()
		}
		if b.l() != nil {
			b.l().Printf("bucket #%d: evict arena %d", b.idx, c)
		}
		wg.Done()
	}()

	wg.Wait()

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
		a := b.arena.head()
		for a != nil {
			a.release()
			a = a.next()
			b.mw().Release(b.ids)
			b.size.snap(snapRelease, b.ac32())
		}
		b.arena.setHead(nil).setAct(nil).setTail(nil)
	}()

	wg.Wait()

	return ErrOK
}

func (b *bucket) flushMetricsLF() {
	var t, u, f uint32
	a := b.arena.head()
	for a != nil {
		t++
		if a.empty() {
			f++
		} else {
			u++
		}
		a = a.next()
	}
	b.mw().ArenaMap(b.ids, t, u, f, b.ac32())
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

func (b *bucket) nowT() time.Time {
	return b.config.Clock.Now()
}

func (b *bucket) alen() uint32 {
	return uint32(b.arena.len())
}

func (b *bucket) elen() uint32 {
	return uint32(len(b.entry))
}

func (b *bucket) mw() MetricsWriter {
	return b.config.MetricsWriter
}

func (b *bucket) as() MemorySize {
	return b.config.ArenaCapacity
}

func (b *bucket) ac32() uint32 {
	return uint32(b.config.ArenaCapacity)
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
