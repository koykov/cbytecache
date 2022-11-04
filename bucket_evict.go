package cbytecache

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

func (b *bucket) evict(e *entry) {
	b.size.snap(snapEvict, e.length)
	b.mw().Free(e.length)
	b.mw().Evict()
	delete(b.index, e.hash)
}
