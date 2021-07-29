package cbytecache

import (
	"fmt"
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

func NewCByteCache(config Config) (*CByteCache, error) {
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
		config: &config,
		status: cacheStatusActive,
		mask:   uint64(config.Shards - 1),
		nowPtr: &now,
	}
	c.shards = make([]*shard, config.Shards)
	for i := range c.shards {
		c.shards[i] = &shard{
			config:  &config,
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

	return c, ErrOK
}

func (c *CByteCache) Set(key string, data []byte) error {
	if err := c.checkCache(); err != nil {
		return err
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
