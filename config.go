package cbytecache

import "time"

type Config struct {
	Shards   uint
	ForceSet bool
	Expire   time.Duration
	Vacuum   time.Duration
	MaxSize  MemorySize
}
