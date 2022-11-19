package cbytecache

import "sync/atomic"

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
	arena := b.arena.act().next()
	for arena != nil {
		arena.release()
		prev := arena
		arena = arena.next()
		prev.setNext(nil).setPrev(nil)
		b.mw().Release(b.ac32())
		b.size.snap(snapRelease, b.ac32())
		c++
	}
	b.arena.act().setNext(nil)
	if b.l() != nil {
		b.l().Printf("bucket #%d: vacuum arena len %d", b.idx, c)
	}

	return ErrOK
}
