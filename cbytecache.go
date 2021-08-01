package cbytecache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/koykov/cbytebuf"
)

type CByteCache struct {
	config *Config
	status uint32
	shards []*shard
	mask   uint64
	nowPtr *uint32
}

func NewCByteCache(config *Config) (*CByteCache, error) {
	config = config.Copy()

	if config.HashFn == nil {
		return nil, ErrBadHashFn
	}
	if (config.Shards & (config.Shards - 1)) != 0 {
		return nil, ErrBadShards
	}
	if uint64(config.MaxSize)/uint64(config.Shards) > MaxShardSize {
		return nil, fmt.Errorf("%d shards on %d cache size exceeds max shard size %d. Reduce cache size or increase shards count",
			config.Shards, config.MaxSize, MaxShardSize)
	}
	if config.Vacuum > 0 && config.Expire > 0 && config.Vacuum <= config.Expire {
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
		mask:   uint64(config.Shards - 1),
		nowPtr: &now,
	}
	c.shards = make([]*shard, config.Shards)
	for i := range c.shards {
		c.shards[i] = &shard{
			config:  config,
			maxSize: uint32(uint64(config.MaxSize) / uint64(config.Shards)),
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
	if err := c.checkCache(); err != nil {
		return err
	}
	if len(data) == 0 {
		return ErrEntryEmpty
	}
	if len(data) > MaxEntrySize {
		return ErrEntryTooBig
	}
	h := c.config.HashFn(key)
	shard := c.shards[h&c.mask]
	return shard.set(h, data)
}

func (c *CByteCache) SetMarshallerTo(key string, m MarshallerTo) error {
	if err := c.checkCache(); err != nil {
		return err
	}
	if m.Size() == 0 {
		return ErrEntryEmpty
	}
	if m.Size() > MaxEntrySize {
		return ErrEntryTooBig
	}
	h := c.config.HashFn(key)
	shard := c.shards[h&c.mask]
	return shard.setm(h, m)
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	return c.GetTo(nil, key)
}

func (c *CByteCache) GetTo(dst []byte, key string) ([]byte, error) {
	if err := c.checkCache(); err != nil {
		return dst, err
	}
	h := c.config.HashFn(key)
	shard := c.shards[h&c.mask]
	return shard.get(dst, h)
}

func (c *CByteCache) evict() error {
	if err := c.checkCache(); err != nil {
		return err
	}
	count := min(evictWorkers, uint32(c.config.Shards))
	shardQueue := make(chan uint, count)
	var wg sync.WaitGroup
	for i := uint32(0); i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var (
				idx uint
				ok  = true
			)
			for ok {
				if idx, ok = <-shardQueue; ok {
					shard := c.shards[idx]
					_ = shard
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := uint(0); i < c.config.Shards; i++ {
			shardQueue <- i
		}
		close(shardQueue)
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
