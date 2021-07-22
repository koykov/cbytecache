package cbytecache

import (
	"time"
)

type Config struct {
	HashFn   func(string) uint64
	Shards   uint
	ForceSet bool
	Expire   time.Duration
	Vacuum   time.Duration
	MaxSize  MemorySize

	MetricsWriter MetricsWriter
	Logger        Logger
}
