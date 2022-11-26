package cbytecache

import (
	"math"
	"sync/atomic"
)

const (
	VacuumRatioWeak       = .25
	VacuumRatioModerate   = .5
	VacuumRatioAggressive = .75
)

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
		b.lastVac = b.nowT()
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
	}()

	if b.nowT().Sub(b.lastEvc) > b.config.EvictInterval/10*9 {
		if err := b.bulkEvictLF(); err != nil {
			return err
		}
	}

	var c int
	a := b.arena.act().next()
	for a != nil {
		a = a.next()
		c++
	}
	r := int(math.Floor(float64(c) * b.config.VacuumRatio))
	a = b.arena.tail()
	c = 0
	for c < r {
		if !a.released() {
			a.release()
			b.mw().Release(b.ids, b.ac32())
			b.mw().ArenaRelease(b.ids)
		}
		tail := a
		a = a.prev()
		tail.setNext(nil).setPrev(nil)
		a.setNext(nil)
		b.size.snap(snapRelease, b.ac32())
		c++
	}
	b.arena.setTail(a)
	if b.l() != nil {
		b.l().Printf("bucket #%d: vacuum arena len %d", b.idx, c)
	}

	return ErrOK
}

var _, _, _ = VacuumRatioWeak, VacuumRatioModerate, VacuumRatioAggressive
