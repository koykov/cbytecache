package cbytecache

import "time"

// MetricsWriter is the interface that wraps the basic metrics methods.
type MetricsWriter interface {
	// Alloc registers how many arenas allocates and increase cache size (total and free).
	Alloc(bucket string, size uint32)
	// Fill registers how many arenas fills and calculate cache size (inc fill and dec free).
	Fill(bucket string, size uint32)
	// Reset registers how many arenas resets and calculate cache size (inc free and dec used).
	Reset(bucket string, size uint32)
	// Release registers how many arenas releases and calculate cache size (dec total and free).
	Release(bucket string, size uint32)
	// Set registers how many entries writes to cache.
	Set(bucket string, dur time.Duration)
	// Hit registers how many entries reads from cache.
	Hit(bucket string, dur time.Duration)
	// Evict registers how many entries evicts from cache.
	Evict(bucket string)
	// Miss registers how many reads failed due to not found error.
	Miss(bucket string)
	// Expire registers how many entries in cache marks as expired.
	Expire(bucket string)
	// Corrupt registers how many entries reads failed due to corruption error.
	Corrupt(bucket string)
	// Collision registers how many keys collisions triggers due to hasher.
	Collision(bucket string)
	// NoSpace registers how many writes failed due to no space error.
	NoSpace(bucket string)
	// Dump registers how many entries dumped.
	Dump()
	// Load registers how many entries loaded from dump.
	Load()
}
