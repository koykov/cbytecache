package cbytecache

import (
	"time"

	"github.com/koykov/hash"
)

// Config describes cache properties and behavior.
type Config struct {
	// Keys hasher helper.
	// Mandatory param.
	Hasher hash.Hasher
	// Buckets count. Must be power of two.
	// Mandatory param.
	Buckets uint
	// Time after which entry will evict.
	Expire time.Duration
	// Time after which entry will flush.
	Vacuum time.Duration
	// Collision checks switch.
	CollisionCheck bool
	// Maximum cache payload size (it doesn't consider index size).
	MaxSize MemorySize
	// Clock implementation.
	Clock Clock

	// Metrics writer handler.
	MetricsWriter MetricsWriter
	// Logger is a logging interface to display verbose messages.
	Logger Logger
}

// Copy copies config instance to protect cache from changing params after start.
// All config modifications will have no effect after copy.
func (c *Config) Copy() *Config {
	cpy := *c
	return &cpy
}

// DefaultConfig makes config with default params.
func DefaultConfig(expire time.Duration, hasher hash.Hasher) *Config {
	c := Config{
		Hasher:  hasher,
		Buckets: 1024,
		Expire:  expire,
	}
	return &c
}

// DefaultConfigWS makes default config with given max size.
func DefaultConfigWS(expire time.Duration, hasher hash.Hasher, maxSize MemorySize) *Config {
	c := DefaultConfig(expire, hasher)
	c.MaxSize = maxSize
	return c
}

var _ = DefaultConfigWS
