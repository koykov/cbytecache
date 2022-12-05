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
func New(conf *Config) (*Cache, error) {
	if conf == nil {
		return nil, ErrBadConfig
	}
	// Config protection.
	conf = conf.Copy()

	// Check mandatory params.
	if conf.Hasher == nil {
		return nil, ErrBadHasher
	}
	if conf.Buckets == 0 || (conf.Buckets&(conf.Buckets-1)) != 0 {
		return nil, ErrBadBuckets
	}
	// Check single bucket size.
	bktCap := uint64(conf.Capacity) / uint64(conf.Buckets)
	if bktCap > 0 && bktCap > MaxBucketSize {
		return nil, fmt.Errorf("%d buckets on %d cache size exceeds max bucket size %d. Reduce cache size or increase buckets count",
			conf.Buckets, conf.Capacity, MaxBucketSize)
	}
	if conf.ArenaCapacity == 0 {
		conf.ArenaCapacity = defaultArenaCapacity
	}
	if bktCap > 0 && bktCap < uint64(conf.ArenaCapacity) {
		return nil, fmt.Errorf("bucket size must be greater than arena size %d", conf.ArenaCapacity)
	}
	// Check expire interval.
	if conf.ExpireInterval < MinExpireInterval {
		return nil, ErrExpireDur
	}
	// Check evict interval.
	if conf.EvictInterval == 0 {
		conf.EvictInterval = conf.ExpireInterval
	}
	// Check vacuum interval.
	if conf.VacuumInterval > 0 && conf.VacuumInterval <= conf.EvictInterval {
		return nil, ErrVacuumDur
	}
	if r := conf.VacuumRatio; r <= 0 || r > 1 {
		conf.VacuumRatio = VacuumRatioModerate
	}

	if conf.MetricsWriter == nil {
		conf.MetricsWriter = &DummyMetrics{}
	}

	if conf.Clock == nil {
		conf.Clock = &NativeClock{}
	}
	if !conf.Clock.Active() {
		conf.Clock.Start()
	}

	// Init the cache and buckets.
	c := &Cache{
		config: conf,
		status: cacheStatusActive,
		mask:   uint64(conf.Buckets - 1),

		maxEntrySize: uint32(bktCap),
	}
	c.buckets = make([]*bucket, conf.Buckets)
	for i := range c.buckets {
		c.buckets[i] = newBucket(uint32(i), conf, bktCap)
	}

	// Register evict schedule job.
	if conf.EvictInterval > 0 {
		if conf.EvictWorkers == 0 {
			conf.EvictWorkers = defaultEvictWorkers
		}
		conf.Clock.Schedule(conf.EvictInterval, func() {
			if err := c.evict(); err != nil && c.l() != nil {
				c.l().Printf("eviction failed with error %s\n", err.Error())
			}
		})
	}
	// Register vacuum schedule job.
	if conf.VacuumInterval > 0 {
		if conf.VacuumWorkers == 0 {
			conf.VacuumWorkers = defaultVacuumWorkers
		}
		conf.Clock.Schedule(conf.VacuumInterval, func() {
			if err := c.vacuum(); err != nil && c.l() != nil {
				c.l().Printf("vacuum failed with error %s\n", err.Error())
			}
		})
	}
	// Register dump schedule job.
	if conf.DumpInterval > 0 {
		if conf.DumpWriteWorkers == 0 {
			conf.DumpWriteWorkers = defaultDumpWriteWorkers
		}
		conf.Clock.Schedule(conf.DumpInterval, func() {
			if err := c.dump(); err != nil && c.l() != nil {
				c.l().Printf("dump write failed with error %s\n", err.Error())
			}
		})
	}

	// Process dumps.
	if conf.DumpReader != nil {
		if conf.DumpReadWorkers == 0 {
			conf.DumpReadWorkers = defaultDumpReadWorkers
		}
		if conf.DumpReadBuffer == 0 {
			conf.DumpReadBuffer = conf.DumpReadWorkers
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
	bkt := c.buckets[h&c.mask]
	return bkt.set(key, h, data)
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
	bkt := c.buckets[h&c.mask]
	return bkt.setm(key, h, m)
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
	bkt := c.buckets[h&c.mask]
	return bkt.get(dst, h)
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

// Evict expired cache data.
func (c *Cache) evict() error {
	return c.bulkExec(c.config.EvictWorkers, "eviction", func(b *bucket) error { return b.bulkEvict() })
}

// Vacuum free cache space.
func (c *Cache) vacuum() error {
	return c.bulkExec(c.config.VacuumWorkers, "vacuum", func(b *bucket) error { return b.bulkVacuum() })
}

// Dump all cache data.
func (c *Cache) dump() error {
	if c.config.DumpWriter == nil {
		return ErrOK
	}
	if err := c.bulkExec(c.config.DumpWriteWorkers, "dump", func(b *bucket) error { return b.bulkDump() }); err != nil {
		return err
	}
	return c.config.DumpWriter.Flush()
}

// Load dumped data.
func (c *Cache) load() error {
	stream := make(chan Entry, c.config.DumpReadBuffer)
	var wg sync.WaitGroup
	for i := uint(0); i < c.config.DumpReadWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case e, ok := <-stream:
					if !ok {
						return
					}
					h := c.config.Hasher.Sum64(e.Key)
					bkt := c.buckets[h&c.mask]
					bkt.mux.Lock()
					_ = bkt.setLF(e.Key, h, e.Body, e.Expire)
					bkt.mux.Unlock()
					c.mw().Load()
				}
			}
		}()
	}

	for {
		e, err := c.config.DumpReader.Read()
		if err == io.EOF {
			close(stream)
			break
		}
		stream <- e.Copy()
	}

	wg.Wait()

	return nil
}

// Perform bulk fn asynchronously.
func (c *Cache) bulkExec(workers uint, op string, fn func(*bucket) error) error {
	return c.bulkExecWS(workers, op, fn, cacheStatusActive)
}

// Perform bulk fn asynchronously with status allow mask.
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
					bkt := c.buckets[idx]
					if err := fn(bkt); err != nil && c.l() != nil {
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

// Check cache status.
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

// Shorthand metrics writer method.
func (c *Cache) mw() MetricsWriter {
	return c.config.MetricsWriter
}

// Shorthand logger method.
func (c *Cache) l() Logger {
	return c.config.Logger
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
