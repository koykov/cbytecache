package cbytecache

import (
	"errors"
	"fmt"
)

var (
	ErrOK error = nil

	ErrBadConfig      = errors.New("config is empty")
	ErrBadCache       = errors.New("cache uninitialized, use New()")
	ErrCacheClosed    = errors.New("cache closed")
	ErrBadHasher      = errors.New("you must provide hasher helper")
	ErrBadBuckets     = errors.New("buckets count must be greater than zero")
	ErrKeyTooBig      = fmt.Errorf("key overflows maximum %d", MaxKeySize)
	ErrNotFound       = errors.New("entry not found")
	ErrEntryExists    = errors.New("entry already exists")
	ErrEntryTooBig    = errors.New("entry too big")
	ErrEntryEmpty     = errors.New("entry is empty")
	ErrEntryCorrupt   = errors.New("entry corrupted")
	ErrEntryCollision = errors.New("entry keys collision")
	ErrExpireDur      = errors.New("expire interval is too short")
	ErrVacuumDur      = errors.New("vacuum interval must be greater than expire interval")
	ErrBucketService  = errors.New("cache bucket is under maintenance")
	ErrBucketCorrupt  = errors.New("cache bucket is corrupted")
	ErrNoSpace        = errors.New("no space available")
	ErrNoEnqueuer     = errors.New("no enqueuer provided")
	ErrNoUnmarshaller = errors.New("no unmarshaller provided")
)
