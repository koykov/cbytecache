package cbytecache

import "errors"

var (
	ErrOK error = nil

	ErrBadShards = errors.New("shards count must be power of two")
	ErrNotFound  = errors.New("entry not found")
)
