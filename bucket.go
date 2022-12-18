package cbytecache

import (
	"encoding/binary"
	"strconv"
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
	queue arenaQueue

	lastEvc, lastVac time.Time
}

// Make and init new bucket.
func newBucket(id uint32, config *Config, maxCap uint64) *bucket {
	b := bucket{
		config: config,
		idx:    id,
		ids:    strconv.Itoa(int(id)),
		maxCap: uint32(maxCap),
		buf:    cbytebuf.NewCByteBuf(),
		index:  make(map[uint64]uint32),
	}
	b.queue.setHead(nil).setAct(nil).setTail(nil)
	return &b
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
	if b.queue.len() == 0 {
		a := b.queue.alloc(nil, b.ac())
		b.queue.setHead(a).setAct(a)
		b.mw().Alloc(b.ids, b.ac())
		b.size.snap(snapAlloc, b.ac())
	}
	// Get current arena.
	a := b.queue.act()
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
			b.queue.setAct(a)
			b.mw().Fill(b.ids, b.ac())
			// Alloc new arena if needed.
			if a == nil {
				if b.maxCap > 0 && b.alen()*b.ac()+b.ac() > b.maxCap {
					b.queue.setAct(b.queue.tail())
					b.mw().NoSpace(b.ids)
					return ErrNoSpace
				}
				a = b.queue.alloc(prev, b.ac())
				b.mw().Alloc(b.ids, b.ac())
				prev.setNext(a)
				b.queue.setAct(a).setTail(a)
				b.size.snap(snapAlloc, b.ac())
			}
			// Calculate rest of bytes to write.
			mustWrite = min(rest, b.ac())
		}
	}

	// Create and register new entry.
	e1 := entry{
		hash:   h,
		offset: arenaOffset,
		length: pl,
		expire: expire,
		aid:    startArena.id,
		qp:     b.queue.ptr(),
	}
	if e1.expire == 0 {
		e1.expire = uint32(b.config.Clock.Now().Add(b.config.ExpireInterval).Unix())
	}
	b.entry = append(b.entry, e1)
	b.index[h] = b.elen() - 1

	b.size.snap(snapSet, pl)
	b.mw().Set(b.ids, b.nowT().Sub(stm))
	return ErrOK
}

// Get entry by h hash.
func (b *bucket) get(dst []byte, h uint64, del bool) ([]byte, error) {
	if err := b.checkStatus(); err != nil {
		return dst, err
	}

	if del {
		b.mux.Lock()
		defer b.mux.Unlock()
	} else {
		b.mux.RLock()
		defer b.mux.RUnlock()
	}
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

	if del {
		err = b.delLF(h)
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

	arenaRest := b.ac() - arenaOffset
	if entry.offset+entry.length < b.ac() {
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
			arenaRest = min(rest, b.ac())
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

// Delete entry from bucket index.
//
// Entry data will keep in the arenas and will
func (b *bucket) del(h uint64) error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	b.mux.Lock()
	defer b.mux.Unlock()

	return b.delLF(h)
}

// Delete entry in lock-free mode.
func (b *bucket) delLF(h uint64) error {
	delete(b.index, h)
	return nil
}

// Reset bucket data.
//
// All allocated data and buffer will keep for further use.
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

// Release bucket data.
//
// All allocated data (arenas and buffer) will release.
func (b *bucket) release() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	var c int
	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: release %d arenas", b.idx, c)
		}
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
		a := b.queue.head()
		for a != nil {
			if a.full() {
				a.reset()
				b.mw().Reset(b.ids, b.ac())
			}
			c++
			a.release()
			b.mw().Release(b.ids, b.ac())
			b.size.snap(snapRelease, b.ac())
			a = a.next()
		}
		b.queue.setHead(nil).setAct(nil).setTail(nil)
	}()

	wg.Wait()

	return ErrOK
}

// Check bucket status.
//
// Possible errors are: bucket under service or bucket corrupt.
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

// Return timestamp as uint32 value.
func (b *bucket) now() uint32 {
	return uint32(b.config.Clock.Now().Unix())
}

// Return current time.
func (b *bucket) nowT() time.Time {
	return b.config.Clock.Now()
}

// Return length of currently used arenas.
func (b *bucket) alen() uint32 {
	return uint32(b.queue.len())
}

// Return length of collected entries.
func (b *bucket) elen() uint32 {
	return uint32(len(b.entry))
}

// Shorthand metrics writer method.
func (b *bucket) mw() MetricsWriter {
	return b.config.MetricsWriter
}

// Shorthand arena capacity method.
func (b *bucket) ac() uint32 {
	return uint32(b.config.ArenaCapacity)
}

// Shorthand logger method.
func (b *bucket) l() Logger {
	return b.config.Logger
}
