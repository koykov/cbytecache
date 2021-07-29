package cbytecache

import "errors"

var (
	ErrOK error = nil

	ErrBadCache     = errors.New("cache uninitialized, use NewCByteCache()")
	ErrCacheClosed  = errors.New("cache closed")
	ErrBadHashFn    = errors.New("you must provide hash function")
	ErrBadShards    = errors.New("shards count must be power of two")
	ErrNotFound     = errors.New("entry not found")
	ErrEntryTooBig  = errors.New("entry too big")
	ErrEntryCorrupt = errors.New("entry corrupted")
	ErrVacuumDur    = errors.New("vacuum interval must be greater than expire interval")
	ErrShardService = errors.New("cache shard is under maintenance")
	ErrNoSpace      = errors.New("no space available")
)
