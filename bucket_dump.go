package cbytecache

import (
	"sort"
)

// Perform bulk dumping operation.
func (b *bucket) bulkDump() error {
	if err := b.checkStatus(); err != nil {
		return err
	}

	var c int
	b.svcLock()
	defer func() {
		if b.l() != nil {
			b.l().Printf("bucket #%d: dump %d entries", b.idx, c)
		}
		b.buf.ResetLen()
		b.svcUnlock()
	}()

	el := b.elen()
	if el == 0 {
		return ErrOK
	}

	buf := b.entry
	now := b.now()
	_ = buf[el-1]
	z := sort.Search(int(el), func(i int) bool {
		return now <= buf[i].expire
	})

	_ = buf[el-1]
	for i := z; i < int(el); i++ {
		b.dump(&buf[i])
		c++
	}

	return ErrOK
}

// Perform dump operation over single entry.
func (b *bucket) dump(e *entry) {
	b.buf.ResetLen()
	_ = b.buf.GrowLen(int(e.length))
	key, body, err := b.getLF(b.buf.Bytes()[:0], e, dummyMetrics)
	if err != nil {
		return
	}
	_, _ = b.config.DumpWriter.Write(Entry{Key: key, Body: body, Expire: e.expire})
	b.mw().Dump(b.ids)
}
