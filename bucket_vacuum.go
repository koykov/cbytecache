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

	var c int
	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: vacuum %d arenas", b.idx, c)
		}
		b.lastVac = b.nowT()
		b.flushMetricsLF()
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
	}()

	if _, _, err := b.bulkEvictLF(true); err != nil {
		return err
	}

	var t int
	a := b.arena.act().next()
	for a != nil {
		if !a.released() {
			t++
		}
		a = a.next()
	}
	r := int(math.Floor(float64(t) * b.config.VacuumRatio))
	a = b.arena.tail()
	for c < r {
		if !a.released() {
			a.release()
			b.mw().Release(b.ids)
		}
		tail := a
		a = a.prev()
		tail.setNext(nil).setPrev(nil)
		a.setNext(nil)
		b.size.snap(snapRelease, b.ac32())
		c++
	}
	b.arena.setTail(a)

	return ErrOK
}

var _, _, _ = VacuumRatioWeak, VacuumRatioModerate, VacuumRatioAggressive
