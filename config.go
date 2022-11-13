package cbytecache

import (
	"time"

	"github.com/koykov/hash"
)

// Config describes cache properties and behavior.
type Config struct {
	// Capacity represents maximum cache payload size (it doesn't consider index size).
	Capacity MemorySize
	// ArenaCapacity determines fixed memory arena size.
	// If this param omit defaultArenaCapacity (16KB) will use instead.
	ArenaCapacity MemorySize

	// Hasher calculates uint64 hash of entries keys.
	// Mandatory param.
	Hasher hash.Hasher
	// Buckets represents buckets (data shards) count. Must be power of two.
	// Mandatory param.
	Buckets uint
	// ExpireInterval represents entry lifetime period.
	// After this period entry may be evicted in any time. Reading of entry after that period will fail as expired.
	// Mandatory param.
	ExpireInterval time.Duration

	// EvictInterval represents period between eviction operations.
	// If this param omit ExpireInterval will use instead.
	EvictInterval time.Duration
	// EvictWorkers limits workers count for evict operation.
	// If this param omit defaultEvictWorkers (16) will use instead.
	EvictWorkers uint

	// VacuumInterval represents period between vacuum operations.
	VacuumInterval time.Duration
	// VacuumWorkers limits workers count for vacuum operation.
	// If this param omit defaultVacuumWorkers (16) will use instead.
	VacuumWorkers uint

	// ResetWorkers limits workers count for reset operation.
	// If this param omit defaultResetWorkers (16) will use instead.
	ResetWorkers uint
	// ReleaseWorkers limits workers count for release operation.
	// If this param omit defaultReleaseWorkers (16) will use instead.
	ReleaseWorkers uint

	// CollisionCheck enables collision checks.
	CollisionCheck bool

	// Clock implementation.
	// If this param omit nativeClock{} will use instead.
	Clock Clock

	// ExpireListener triggers on every expired item.
	ExpireListener Listener

	// DumpWriter represents writer for dumps.
	DumpWriter DumpWriter
	// DumpInterval indicates how often need dump cache data.
	DumpInterval time.Duration
	// DumpWriteWorkers limits workers count that sends entries to DumpWriter.
	// If this param omit defaultDumpWriteWorkers (16) will use instead.
	DumpWriteWorkers uint

	// DumpReader represents dump loader that fills cache with dumped data.
	DumpReader DumpReader
	// DumpReadBuffer represents how many items from dump may be processed at once.
	DumpReadBuffer uint
	// DumpReadWorkers limits workers count that processes entries comes from DumpReader.
	// If this param omit defaultDumpReadWorkers (16) will use instead.
	DumpReadWorkers uint

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
