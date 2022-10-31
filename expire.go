package cbytecache

func (b *bucket) expireRange(z int) {
	el := b.elen()
	if z < 256 {
		_ = b.entry[el-1]
		for i := 0; i < z; i++ {
			b.expire(&b.entry[i])
		}
	} else {
		z8 := z - z%8
		_ = b.entry[el-1]
		for i := 0; i < z8; i += 8 {
			b.expire(&b.entry[i])
			b.expire(&b.entry[i+1])
			b.expire(&b.entry[i+2])
			b.expire(&b.entry[i+3])
			b.expire(&b.entry[i+4])
			b.expire(&b.entry[i+5])
			b.expire(&b.entry[i+6])
			b.expire(&b.entry[i+7])
		}
		for i := z8; i < z; i++ {
			b.expire(&b.entry[i])
		}
	}
}

func (b *bucket) expire(e *entry) {
	b.buf.ResetLen()
	_ = b.buf.GrowLen(int(e.length))
	key, body, err := b.getLF(b.buf.Bytes(), e, dummyMetrics)
	if err != nil {
		return
	}
	_ = b.config.ExpireListener.Listen(key, body)
}
