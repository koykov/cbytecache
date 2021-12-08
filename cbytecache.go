package cbytecache

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/koykov/cbytebuf"
)

type CByteCache struct {
	config  *Config
	status  uint32
	buckets []*bucket
	mask    uint64

	maxEntrySize uint32
}

func NewCByteCache(config *Config) (*CByteCache, error) {
	if config == nil {
		return nil, ErrBadConfig
	}
	config = config.Copy()

	if config.Hasher == nil {
		return nil, ErrBadHasher
	}
	if config.Buckets == 0 || (config.Buckets&(config.Buckets-1)) != 0 {
		return nil, ErrBadBuckets
	}
	bucketSize := uint64(config.MaxSize) / uint64(config.Buckets)
	if bucketSize > 0 && bucketSize > MaxBucketSize {
		return nil, fmt.Errorf("%d buckets on %d cache size exceeds max bucket size %d. Reduce cache size or increase buckets count",
			config.Buckets, config.MaxSize, MaxBucketSize)
	}
	if bucketSize > 0 && bucketSize < uint64(ArenaSize) {
		return nil, fmt.Errorf("bucket size must be greater than arena size %d", ArenaSize)
	}
	if config.Expire > 0 && config.Expire < MinExpireInterval {
		return nil, ErrExpireDur
	}
	if config.Vacuum > 0 && config.Vacuum <= config.Expire {
		return nil, ErrVacuumDur
	}

	if config.MetricsWriter == nil {
		config.MetricsWriter = &DummyMetrics{}
	}

	if config.Logger == nil {
		config.Logger = dummyLog
	}

	if config.Clock == nil {
		config.Clock = &NativeClock{}
	}
	if !config.Clock.Active() {
		config.Clock.Start()
	}

	c := &CByteCache{
		config: config,
		status: cacheStatusActive,
		mask:   uint64(config.Buckets - 1),

		maxEntrySize: uint32(bucketSize),
	}
	c.buckets = make([]*bucket, config.Buckets)
	for i := range c.buckets {
		c.buckets[i] = &bucket{
			config:  config,
			idx:     uint32(i),
			maxSize: uint32(bucketSize),
			buf:     cbytebuf.NewCByteBuf(),
			index:   make(map[uint64]uint32),
		}
	}

	if config.Expire > 0 {
		config.Clock.Schedule(config.Expire, func() {
			if err := c.evict(); err != nil && c.config.Logger != nil {
				c.config.Logger.Printf("eviction failed with error %s\n", err.Error())
			}
		})
	}
	if config.Vacuum > 0 {
		config.Clock.Schedule(config.Vacuum, func() {
			if err := c.vacuum(); err != nil && c.config.Logger != nil {
				c.config.Logger.Printf("vacuum failed with error %s\n", err.Error())
			}
		})
	}

	return c, ErrOK
}

func (c *CByteCache) Set(key string, data []byte) error {
	return c.set(key, data)
}

func (c *CByteCache) SetMarshallerTo(key string, m MarshallerTo) error {
	return c.setm(key, m)
}

func (c *CByteCache) set(key string, data []byte) error {
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

func (c *CByteCache) setm(key string, m MarshallerTo) error {
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

func (c *CByteCache) Get(key string) ([]byte, error) {
	return c.GetTo(nil, key)
}

func (c *CByteCache) GetTo(dst []byte, key string) ([]byte, error) {
	if err := c.checkCache(cacheStatusActive); err != nil {
		return dst, err
	}
	h := c.config.Hasher.Sum64(key)
	bucket := c.buckets[h&c.mask]
	return bucket.get(dst, h)
}

func (c *CByteCache) Reset() error {
	return c.bulkExec(resetWorkers, "reset", func(b *bucket) error { return b.reset() })
}

func (c *CByteCache) Release() error {
	return c.bulkExecWS(releaseWorkers, "release", func(b *bucket) error { return b.release() }, cacheStatusActive|cacheStatusClosed)
}

func (c *CByteCache) Close() error {
	atomic.StoreUint32(&c.status, cacheStatusClosed)
	if err := c.Release(); err != nil {
		return err
	}
	c.config.Clock.Stop()
	return ErrOK
}

func (c *CByteCache) evict() error {
	return c.bulkExec(evictWorkers, "eviction", func(b *bucket) error { return b.bulkEvict() })
}

func (c *CByteCache) vacuum() error {
	return c.bulkExec(evictWorkers, "vacuum", func(b *bucket) error { return b.vacuum() })
}

func (c *CByteCache) bulkExec(workers int, op string, fn func(*bucket) error) error {
	return c.bulkExecWS(workers, op, fn, cacheStatusActive)
}

func (c *CByteCache) bulkExecWS(workers int, op string, fn func(*bucket) error, allow uint32) error {
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
					if err := fn(bucket); err != nil && c.config.Logger != nil {
						c.config.Logger.Printf("bucket %d %s failed with error: %s\n", idx, op, err.Error())
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

func (c *CByteCache) checkCache(allow uint32) error {
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
