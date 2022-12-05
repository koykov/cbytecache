package cbytecache

import (
	"math"
	"sync/atomic"
)

// Bucket size snapshot type.
type snap int

const (
	snapAlloc snap = iota
	snapSet
	snapEvict
	snapRelease
)

// Collection of bucket sizes.
type bucketSize struct {
	total, used, free uint32
}

// Collect size for given snapshot type.
func (s *bucketSize) snap(op snap, size uint32) {
	switch op {
	case snapAlloc:
		atomic.AddUint32(&s.total, size)
		atomic.AddUint32(&s.free, size)
	case snapSet:
		atomic.AddUint32(&s.used, size)
		atomic.AddUint32(&s.free, math.MaxUint32-size+1)
	case snapEvict:
		atomic.AddUint32(&s.used, math.MaxUint32-size+1)
		atomic.AddUint32(&s.free, size)
	case snapRelease:
		atomic.AddUint32(&s.total, math.MaxUint32-size+1)
		atomic.AddUint32(&s.free, math.MaxUint32-size+1)
	}
}

// Get collected size snapshot data.
func (s *bucketSize) snapshot() (uint32, uint32, uint32) {
	return atomic.LoadUint32(&s.total), atomic.LoadUint32(&s.used), atomic.LoadUint32(&s.free)
}
