package cbytecache

import (
	"math"
)

const (
	VacuumRatioWeak       = .25
	VacuumRatioModerate   = .5
	VacuumRatioAggressive = .75
)

// Perform bilk vacuum operation.
func (b *bucket) bulkVacuum() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	var c int
	b.svcLock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: vacuum %d arenas", b.idx, c)
		}
		b.lastVac = b.nowT()
		b.svcUnlock()
	}()

	// Run eviction before dump to evaluate number of arenas possible to vacuum.
	if _, _, err := b.bulkEvictLF(true); err != nil {
		return err
	}

	var t int
	a := b.queue.act().next()
	for a != nil {
		if !a.released() {
			t++
		}
		a = a.next()
	}
	r := int(math.Floor(float64(t) * b.config.VacuumRatio))

	// Vacuum r arenas starting from tail.
	a = b.queue.tail()
	for c < r {
		if !a.released() {
			a.release()
			b.mw().Release(b.ids, b.ac())
		}
		tail := a
		a = a.prev()
		tail.setNext(nil).setPrev(nil)
		a.setNext(nil)
		b.size.snap(snapRelease, b.ac())
		c++
	}
	// Register last arena as tail.
	b.queue.setTail(a)

	return ErrOK
}

var _, _, _ = VacuumRatioWeak, VacuumRatioModerate, VacuumRatioAggressive
