package cbytecache

import (
	"sort"
	"sync"
)

// Perform bulk eviction operation.
func (b *bucket) bulkEvict() (err error) {
	if err = b.checkStatus(); err != nil {
		return
	}

	var ac, ec int
	b.svcLock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: evict %d entries and free up %d arenas", b.idx, ec, ac)
		}
		b.svcUnlock()
	}()

	ac, ec, err = b.bulkEvictLF(false)
	return
}

// Perform bulk eviction operation on lock-free mode.
func (b *bucket) bulkEvictLF(force bool) (ac, ec int, err error) {
	err = ErrOK
	if !force && b.nowT().Sub(b.lastEvc) < b.config.EvictInterval/10*9 {
		return
	}

	defer func() {
		b.lastEvc = b.nowT()
	}()

	el := b.elen()
	if el == 0 {
		return
	}

	buf := b.entry
	now := b.now()
	_ = buf[el-1]
	z := sort.Search(int(el), func(i int) bool {
		return now <= buf[i].expire
	})
	if z == 0 {
		return
	}
	ec = z

	// Last arena contains unexpired entries.
	lo := buf[z-1].arena()
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
		wg.Done()
	}()

	// Recycle arenas.
	wg.Add(1)
	go func() {
		b.queue.recycle(lo1)
		// Reset all arenas after actual.
		a := b.queue.act().next()
		for a != nil {
			if !a.empty() {
				a.reset()
				b.mw().Reset(b.ids, b.acap())
				ac++
			}
			a = a.next()
		}
		wg.Done()
	}()

	wg.Wait()

	return
}

// Evict all entries on range [0..z).
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

	// Move non-expired entries to the start of entries list.
	copy(b.entry, b.entry[z:])
	b.entry = b.entry[:el-uint32(z)]

	// Update index.
	if el = b.elen(); el == 0 {
		return
	}
	if el < 256 {
		_ = b.entry[el-1]
		for i := uint32(0); i < el; i++ {
			b.index[b.entry[i].hash] = i
		}
	} else {
		el8 := el - el%8
		for i := uint32(0); i < el8; i += 8 {
			b.index[b.entry[i].hash] = i
			b.index[b.entry[i+1].hash] = i + 1
			b.index[b.entry[i+2].hash] = i + 2
			b.index[b.entry[i+3].hash] = i + 3
			b.index[b.entry[i+4].hash] = i + 4
			b.index[b.entry[i+5].hash] = i + 5
			b.index[b.entry[i+6].hash] = i + 6
			b.index[b.entry[i+7].hash] = i + 7
		}
		for i := el8; i < el; i++ {
			b.index[b.entry[i].hash] = i
		}
	}
}

// Perform evict operation over single entry.
func (b *bucket) evict(e *entry) {
	b.size.snap(snapEvict, e.length)
	b.mw().Evict(b.ids, !e.invalid())
	if e.invalid() {
		return
	}
	delete(b.index, e.hash)
}
