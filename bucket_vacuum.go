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
		b.mux.Unlock()
		atomic.StoreUint32(&b.status, bucketStatusActive)
	}()

	for i := b.arendIdx + 1; i < b.alen(); i++ {
		b.arena[i].release()
		b.mw().Release(b.ac32())
		b.size.snap(snapRelease, b.ac32())
	}
	b.arena = b.arena[:b.arendIdx+1]

	return ErrOK
}
