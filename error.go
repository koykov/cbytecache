package cbytecache

import "errors"

var (
	ErrOK error = nil

	ErrBadShards    = errors.New("shards count must be power of two")
	ErrNotFound     = errors.New("entry not found")
	ErrEntryTooBig  = errors.New("entry too big")
	ErrEntryCorrupt = errors.New("entry corrupted")
	ErrVacuumDur    = errors.New("vacuum interval must be greater than expire interval")
)
