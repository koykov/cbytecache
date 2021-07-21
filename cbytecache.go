package cbytecache

import (
	"fmt"

	"github.com/koykov/hash/fnv"
)

type CByteCache struct {
	config *Config
	shards []*shard
	mask   uint64
}

func NewCByteCache(config Config) (*CByteCache, error) {
	if (config.Shards & (config.Shards - 1)) != 0 {
		return nil, ErrBadShards
	}
	if uint64(config.MaxSize)/uint64(config.Shards) > MaxShardSize {
		return nil, fmt.Errorf("%d shards on %d cache size exceeds max shard size %d. Reduce cache size or increase shards count",
			config.Shards, config.MaxSize, MaxShardSize)
	}
	if config.Vacuum <= config.Expire {
		return nil, ErrVacuumDur
	}

	shards := make([]*shard, config.Shards)
	for i := range shards {
		shards[i] = newShard()
	}

	c := &CByteCache{
		config: &config,
		shards: shards,
		mask:   uint64(config.Shards - 1),
	}
	return c, ErrOK
}

func (c *CByteCache) Set(key string, data []byte) error {
	if len(data) > MaxEntrySize {
		return ErrEntryTooBig
	}
	hash := fnv.Hash64aString(key)
	shard := c.shards[hash&c.mask]
	return shard.set(hash, data)
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	return c.GetTo(nil, key)
}

func (c *CByteCache) GetTo(dst []byte, key string) ([]byte, error) {
	hash := fnv.Hash64aString(key)
	shard := c.shards[hash&c.mask]
	return shard.get(dst, hash)
}
