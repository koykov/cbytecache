package cbytecache

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

// Cache is a byte cache implementation based on cbyte package.
type Cache struct {
	config  *Config
	status  uint32
	buckets []*bucket
	mask    uint64

	maxEntrySize uint32
}

// New makes cache instance according config.
func New(config *Config) (*Cache, error) {
	if config == nil {
		return nil, ErrBadConfig
	}
	// Config protection.
	config = config.Copy()

	// Check mandatory params.
	if config.Hasher == nil {
		return nil, ErrBadHasher
	}
	if config.Buckets == 0 || (config.Buckets&(config.Buckets-1)) != 0 {
		return nil, ErrBadBuckets
	}
	// Check single bucket size.
	bucketSize := uint64(config.Capacity) / uint64(config.Buckets)
	if bucketSize > 0 && bucketSize > MaxBucketSize {
		return nil, fmt.Errorf("%d buckets on %d cache size exceeds max bucket size %d. Reduce cache size or increase buckets count",
			config.Buckets, config.Capacity, MaxBucketSize)
	}
	if config.ArenaCapacity == 0 {
		config.ArenaCapacity = defaultArenaCapacity
	}
	if bucketSize > 0 && bucketSize < uint64(config.ArenaCapacity) {
		return nil, fmt.Errorf("bucket size must be greater than arena size %d", config.ArenaCapacity)
	}
	// Check expire interval.
	if config.ExpireInterval < MinExpireInterval {
		return nil, ErrExpireDur
	}
	// Check evict interval.
	if config.EvictInterval == 0 {
		config.EvictInterval = config.ExpireInterval
	}
	// Check vacuum interval.
	if config.VacuumInterval > 0 && config.VacuumInterval <= config.EvictInterval {
		return nil, ErrVacuumDur
	}
	if r := config.VacuumRatio; r <= 0 || r > 1 {
		config.VacuumRatio = VacuumRatioModerate
	}

	if config.MetricsWriter == nil {
		config.MetricsWriter = &DummyMetrics{}
	}

	if config.Clock == nil {
		config.Clock = &NativeClock{}
	}
	if !config.Clock.Active() {
		config.Clock.Start()
	}

	// Init the cache and buckets.
	c := &Cache{
		config: config,
		status: cacheStatusActive,
		mask:   uint64(config.Buckets - 1),

		maxEntrySize: uint32(bucketSize),
	}
	c.buckets = make([]*bucket, config.Buckets)
	for i := range c.buckets {
		c.buckets[i] = newBucket(uint32(i), config, bucketSize)
	}

	// Register evict schedule job.
	if config.EvictInterval > 0 {
		if config.EvictWorkers == 0 {
			config.EvictWorkers = defaultEvictWorkers
		}
		config.Clock.Schedule(config.EvictInterval, func() {
			if err := c.evict(); err != nil && c.l() != nil {
				c.l().Printf("eviction failed with error %s\n", err.Error())
			}
		})
	}
	// Register vacuum schedule job.
	if config.VacuumInterval > 0 {
		if config.VacuumWorkers == 0 {
			config.VacuumWorkers = defaultVacuumWorkers
		}
		config.Clock.Schedule(config.VacuumInterval, func() {
			if err := c.vacuum(); err != nil && c.l() != nil {
				c.l().Printf("vacuum failed with error %s\n", err.Error())
			}
		})
	}
	// Register dump schedule job.
	if config.DumpInterval > 0 {
		if config.DumpWriteWorkers == 0 {
			config.DumpWriteWorkers = defaultDumpWriteWorkers
		}
		config.Clock.Schedule(config.DumpInterval, func() {
			if err := c.dump(); err != nil && c.l() != nil {
				c.l().Printf("dump write failed with error %s\n", err.Error())
			}
		})
	}

	// Process dumps.
	if config.DumpReader != nil {
		if config.DumpReadWorkers == 0 {
			config.DumpReadWorkers = defaultDumpReadWorkers
		}
		if config.DumpReadBuffer == 0 {
			config.DumpReadBuffer = config.DumpReadWorkers
		}
		if err := c.load(); err != nil && c.l() != nil {
			c.l().Printf("dump read failed with error %s\n", err.Error())
		}
	}

	return c, ErrOK
}

// Set sets entry bytes to the cache.
func (c *Cache) Set(key string, data []byte) error {
	return c.set(key, data)
}

// SetMarshallerTo sets entry like protobuf object to the cache.
func (c *Cache) SetMarshallerTo(key string, m MarshallerTo) error {
	return c.setm(key, m)
}

// Internal bytes setter.
func (c *Cache) set(key string, data []byte) error {
	if len(key) > MaxKeySize {
		return ErrKeyTooBig
	}
	if err := c.checkCache(cacheStatusActive); err != nil {
		return err
	}
	dl := uint32(len(data))
	if dl == 0 {
		return ErrEntryEmpty
	}
	if c.maxEntrySize > 0 && dl > c.maxEntrySize {
		return ErrEntryTooBig
	}
	h := c.config.Hasher.Sum64(key)
	bucket := c.buckets[h&c.mask]
	return bucket.set(key, h, data)
}

// Internal marshaller object setter.
func (c *Cache) setm(key string, m MarshallerTo) error {
	if len(key) > MaxKeySize {
		return ErrKeyTooBig
	}
	if err := c.checkCache(cacheStatusActive); err != nil {
		return err
	}
	ml := uint32(m.Size())
	if ml == 0 {
		return ErrEntryEmpty
	}
	if ml > c.maxEntrySize {
		return ErrEntryTooBig
	}
	h := c.config.Hasher.Sum64(key)
	bucket := c.buckets[h&c.mask]
	return bucket.setm(key, h, m)
}

// Get gets entry bytes by key.
func (c *Cache) Get(key string) ([]byte, error) {
	return c.GetTo(nil, key)
}

// GetTo gets entry bytes to dst.
func (c *Cache) GetTo(dst []byte, key string) ([]byte, error) {
	if err := c.checkCache(cacheStatusActive); err != nil {
		return dst, err
	}
	h := c.config.Hasher.Sum64(key)
	bucket := c.buckets[h&c.mask]
	return bucket.get(dst, h)
}

// Size returns cache size snapshot. Contains total, used and free sizes.
func (c *Cache) Size() (r CacheSize) {
	_ = c.buckets[len(c.buckets)-1]
	for i := 0; i < len(c.buckets); i++ {
		t, u, f := c.buckets[i].size.snapshot()
		r.t += MemorySize(t)
		r.u += MemorySize(u)
		r.f += MemorySize(f)
	}
	return
}

// Reset performs force eviction of all check entries.
func (c *Cache) Reset() error {
	return c.bulkExec(defaultResetWorkers, "reset", func(b *bucket) error { return b.reset() })
}

// Release releases all cache data.
func (c *Cache) Release() error {
	return c.bulkExecWS(defaultReleaseWorkers, "release", func(b *bucket) error { return b.release() }, cacheStatusActive|cacheStatusClosed)
}

// Close destroys cache and releases all data.
//
// You cannot use cache after that.
func (c *Cache) Close() error {
	atomic.StoreUint32(&c.status, cacheStatusClosed)
	if err := c.Release(); err != nil {
		return err
	}
	c.config.Clock.Stop()
	return ErrOK
}

func (c *Cache) evict() error {
	return c.bulkExec(c.config.EvictWorkers, "eviction", func(b *bucket) error { return b.bulkEvict() })
}

func (c *Cache) vacuum() error {
	return c.bulkExec(c.config.VacuumWorkers, "vacuum", func(b *bucket) error { return b.vacuum() })
}

func (c *Cache) dump() error {
	if err := c.bulkExec(c.config.DumpWriteWorkers, "dump", func(b *bucket) error { return b.bulkDump() }); err != nil {
		return err
	}
	return c.config.DumpWriter.Flush()
}

func (c *Cache) load() error {
	stream := make(chan Entry, c.config.DumpReadBuffer)
	var wg sync.WaitGroup
	for i := uint(0); i < c.config.DumpReadWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case entry, ok := <-stream:
					if !ok {
						return
					}
					h := c.config.Hasher.Sum64(entry.Key)
					bucket := c.buckets[h&c.mask]
					bucket.mux.Lock()
					_ = bucket.setLF(entry.Key, h, entry.Body, entry.Expire)
					bucket.mux.Unlock()
					c.mw().Load()
				}
			}
		}()
	}

	for {
		entry, err := c.config.DumpReader.Read()
		if err == io.EOF {
			close(stream)
			break
		}
		stream <- entry.Copy()
	}

	wg.Wait()

	return nil
}

func (c *Cache) bulkExec(workers uint, op string, fn func(*bucket) error) error {
	return c.bulkExecWS(workers, op, fn, cacheStatusActive)
}

func (c *Cache) bulkExecWS(workers uint, op string, fn func(*bucket) error, allow uint32) error {
	if err := c.checkCache(allow); err != nil {
		return err
	}
	count := min(uint32(workers), uint32(c.config.Buckets))
	bucketQueue := make(chan uint, count)
	var wg sync.WaitGroup

	for i := uint32(0); i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if idx, ok := <-bucketQueue; ok {
					bucket := c.buckets[idx]
					if err := fn(bucket); err != nil && c.l() != nil {
						c.l().Printf("bucket #%d: %s failed with error '%s'\n", idx, op, err.Error())
					}
					continue
				}
				break
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := uint(0); i < c.config.Buckets; i++ {
			bucketQueue <- i
		}
		close(bucketQueue)
	}()

	wg.Wait()

	return ErrOK
}

func (c *Cache) checkCache(allow uint32) error {
	if status := atomic.LoadUint32(&c.status); status&allow == 0 {
		if status == cacheStatusNil {
			return ErrBadCache
		}
		if status == cacheStatusClosed {
			return ErrCacheClosed
		}
	}
	return nil
}

func (c *Cache) mw() MetricsWriter {
	return c.config.MetricsWriter
}

func (c *Cache) l() Logger {
	return c.config.Logger
}
