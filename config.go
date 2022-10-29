package cbytecache

import (
	"time"

	"github.com/koykov/hash"
)

// Config describes cache properties and behavior.
type Config struct {
	// Maximum cache payload size (it doesn't consider index size).
	Capacity MemorySize
	// Keys hasher helper.
	// Mandatory param.
	Hasher hash.Hasher
	// Buckets count. Must be power of two.
	// Mandatory param.
	Buckets uint
	// ArenaCapacity determines fixed memory arena size.
	// If this param omit defaultArenaCapacity (1MB) will use instead.
	ArenaCapacity MemorySize
	// Time after which entry will evict.
	ExpireInterval time.Duration
	// Time after which entry will flush.
	VacuumInterval time.Duration
	// Collision checks switch.
	CollisionCheck bool
	// Clock implementation.
	Clock Clock

	ExpireListener Listener
	EvictListener  Listener

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
		Hasher:         hasher,
		Buckets:        16,
		ExpireInterval: expire,
	}
	return &c
}

// DefaultConfigWS makes default config with given capacity.
func DefaultConfigWS(expire time.Duration, hasher hash.Hasher, capacity MemorySize) *Config {
	c := DefaultConfig(expire, hasher)
	c.Capacity = capacity
	return c
}

var _ = DefaultConfigWS
