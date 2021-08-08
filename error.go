package cbytecache

import "errors"

var (
	ErrOK error = nil

	ErrBadConfig     = errors.New("config is empty")
	ErrBadCache      = errors.New("cache uninitialized, use NewCByteCache()")
	ErrCacheClosed   = errors.New("cache closed")
	ErrBadHashFn     = errors.New("you must provide hash function")
	ErrBadBuckets    = errors.New("buckets count must be power of two and great than zero")
	ErrNotFound      = errors.New("entry not found")
	ErrEntryExists   = errors.New("entry already exists")
	ErrEntryTooBig   = errors.New("entry too big")
	ErrEntryEmpty    = errors.New("entry is empty")
	ErrEntryCorrupt  = errors.New("entry corrupted")
	ErrExpireDur     = errors.New("expire interval is too short")
	ErrVacuumDur     = errors.New("vacuum interval must be greater than expire interval")
	ErrBucketService = errors.New("cache bucket is under maintenance")
	ErrNoSpace       = errors.New("no space available")
)
