package cbytecache

import (
	"time"

	"github.com/koykov/hash"
)

type Config struct {
	Key            string
	Hasher         hash.Hasher
	Buckets        uint
	Expire         time.Duration
	Vacuum         time.Duration
	CollisionCheck bool
	MaxSize        MemorySize
	Clock          Clock

	MetricsWriter MetricsWriter
	Logger        Logger
}

func (c *Config) Copy() *Config {
	cpy := *c
	return &cpy
}

func DefaultConfig(key string, expire time.Duration, hasher hash.Hasher) *Config {
	c := Config{
		Key:     key,
		Hasher:  hasher,
		Buckets: 1024,
		Expire:  expire,
	}
	return &c
}

func DefaultConfigWS(key string, expire time.Duration, hasher hash.Hasher, maxSize MemorySize) *Config {
	c := DefaultConfig(key, expire, hasher)
	c.MaxSize = maxSize
	return c
}

var _ = DefaultConfigWS
