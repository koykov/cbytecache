package cbytecache

import (
	"sort"
	"sync/atomic"
)

func (b *bucket) bulkDump() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	if b.l() != nil {
		b.l().Printf("bucket #%d: bulkDump write started", b.idx)
	}

	atomic.StoreUint32(&b.status, bucketStatusService)
	b.mux.Lock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: bulkDump write finished", b.idx)
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

	if z == int(el) {
		return ErrOK
	}

	_ = b.entry[el-1]
	for i := z; i < int(el); i++ {
		b.dump(&b.entry[i])
	}

	return ErrOK
}

func (b *bucket) dump(e *entry) {
	b.buf.ResetLen()
	_ = b.buf.GrowLen(int(e.length))
	key, body, err := b.getLF(b.buf.Bytes()[:0], e, dummyMetrics)
	if err != nil {
		return
	}
	_ = b.config.DumpWriter.Write(Entry{Key: key, Body: body})
	b.mw().Dump()
}
