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

	if z == 0 {
		return ErrOK
	}

	if z < 256 {
		_ = b.entry[el-1]
		for i := 0; i < z; i++ {
			b.dump(&b.entry[i])
		}
	} else {
		z8 := z - z%8
		_ = b.entry[el-1]
		for i := 0; i < z8; i += 8 {
			b.dump(&b.entry[i])
			b.dump(&b.entry[i+1])
			b.dump(&b.entry[i+2])
			b.dump(&b.entry[i+3])
			b.dump(&b.entry[i+4])
			b.dump(&b.entry[i+5])
			b.dump(&b.entry[i+6])
			b.dump(&b.entry[i+7])
		}
		for i := z8; i < z; i++ {
			b.dump(&b.entry[i])
		}
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
}
