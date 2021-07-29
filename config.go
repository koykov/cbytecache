package cbytecache

import (
	"time"
)

type Config struct {
	HashFn  func(string) uint64
	Shards  uint
	Expire  time.Duration
	Vacuum  time.Duration
	MaxSize MemorySize

	MetricsWriter MetricsWriter
	Logger        Logger
}

func (c *Config) Copy() *Config {
	cpy := *c
	return &cpy
}

func DefaultConfig(expire time.Duration, hashFn func(string) uint64) *Config {
	c := Config{
		HashFn: hashFn,
		Shards: 1024,
		Expire: expire,
	}
	return &c
}

func DefaultConfigWS(expire time.Duration, hashFn func(string) uint64, maxSize MemorySize) *Config {
	c := DefaultConfig(expire, hashFn)
	c.MaxSize = maxSize
	return c
}

var _ = DefaultConfigWS
