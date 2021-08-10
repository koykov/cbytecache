package cbytecache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/koykov/cbytebuf"
)

type CByteCache struct {
	config  *Config
	status  uint32
	buckets []*bucket
	mask    uint64
	nowPtr  *uint32

	maxEntrySize uint32
}

func NewCByteCache(config *Config) (*CByteCache, error) {
	if config == nil {
		return nil, ErrBadConfig
	}
	config = config.Copy()

	if config.HashFn == nil {
		return nil, ErrBadHashFn
	}
	if config.Buckets == 0 || (config.Buckets&(config.Buckets-1)) != 0 {
		return nil, ErrBadBuckets
	}
	bucketSize := uint64(config.MaxSize) / uint64(config.Buckets)
	if bucketSize > 0 && bucketSize > MaxBucketSize {
		return nil, fmt.Errorf("%d buckets on %d cache size exceeds max bucket size %d. Reduce cache size or increase buckets count",
			config.Buckets, config.MaxSize, MaxBucketSize)
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
		config.Logger = &DummyLog{}
	}

	now := uint32(time.Now().Unix())
	c := &CByteCache{
		config: config,
		status: cacheStatusActive,
		mask:   uint64(config.Buckets - 1),
		nowPtr: &now,

		maxEntrySize: uint32(bucketSize),
	}
	c.buckets = make([]*bucket, config.Buckets)
	for i := range c.buckets {
		c.buckets[i] = &bucket{
			config:  config,
			maxSize: uint32(bucketSize),
			buf:     cbytebuf.NewCByteBuf(),
			index:   make(map[uint64]uint32),
			nowPtr:  c.nowPtr,
		}
	}

	tickerNow := time.NewTicker(time.Second)
	go func() {
		for {
			select {
			// todo implement done signal.
			case <-tickerNow.C:
				atomic.StoreUint32(c.nowPtr, uint32(time.Now().Unix()))
			}
		}
	}()

	if config.Expire > 0 {
		tickerExpire := time.NewTicker(config.Expire)
		go func() {
			for {
				select {
				// todo implement done signal.
				case <-tickerExpire.C:
					//
				}
			}
		}()
	}

	return c, ErrOK
}

func (c *CByteCache) Set(key string, data []byte) error {
	return c.set(key, data, c.config.ForceSet)
}

func (c *CByteCache) SetMarshallerTo(key string, m MarshallerTo) error {
	return c.setm(key, m, c.config.ForceSet)
}

func (c *CByteCache) FSet(key string, data []byte) error {
	return c.set(key, data, true)
}

func (c *CByteCache) FSetMarshallerTo(key string, m MarshallerTo) error {
	return c.setm(key, m, true)
}

func (c *CByteCache) set(key string, data []byte, force bool) error {
	if err := c.checkCache(); err != nil {
		return err
	}
	dl := uint32(len(data))
	if dl == 0 {
		return ErrEntryEmpty
	}
	if c.maxEntrySize > 0 && dl > c.maxEntrySize {
		return ErrEntryTooBig
	}
	h := c.config.HashFn(key)
	bucket := c.buckets[h&c.mask]
	return bucket.set(key, h, data, force)
}

func (c *CByteCache) setm(key string, m MarshallerTo, force bool) error {
	if err := c.checkCache(); err != nil {
		return err
	}
	ml := uint32(m.Size())
	if ml == 0 {
		return ErrEntryEmpty
	}
	if ml > c.maxEntrySize {
		return ErrEntryTooBig
	}
	h := c.config.HashFn(key)
	bucket := c.buckets[h&c.mask]
	return bucket.setm(key, h, m, force)
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	return c.GetTo(nil, key)
}

func (c *CByteCache) GetTo(dst []byte, key string) ([]byte, error) {
	if err := c.checkCache(); err != nil {
		return dst, err
	}
	h := c.config.HashFn(key)
	bucket := c.buckets[h&c.mask]
	return bucket.get(dst, h)
}

func (c *CByteCache) Reset() error {
	return c.bulkExec(resetWorkers, "reset", func(b *bucket) error { return b.reset() })
}

func (c *CByteCache) evict() error {
	return c.bulkExec(evictWorkers, "eviction", func(b *bucket) error { return b.bulkEvict() })
}

func (c *CByteCache) bulkExec(workers int, op string, fn func(*bucket) error) error {
	if err := c.checkCache(); err != nil {
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

func (c *CByteCache) checkCache() error {
	if status := atomic.LoadUint32(&c.status); status != cacheStatusActive {
		if status == cacheStatusNil {
			return ErrBadCache
		}
		if status == cacheStatusClosed {
			return ErrCacheClosed
		}
	}
	return nil
}
