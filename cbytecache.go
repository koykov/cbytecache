package cbytecache

import "github.com/koykov/hash/fnv"

type CByteCache struct {
	config *Config
	shards []*shard
	mask   uint32
	Alg    uint
}

func NewCByteCache(config Config) *CByteCache {
	shards := make([]*shard, config.Shards)
	for i := range shards {
		shards[i] = newShard()
	}

	c := &CByteCache{
		config: &config,
		shards: shards,
		mask:   uint32(config.Shards - 1),
	}
	return c
}

func (c *CByteCache) Set(key string, data []byte) error {
	hash := fnv.Hash32aString(key)
	shard := c.shards[hash&c.mask]
	return shard.set(hash, data)
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	return c.GetTo(nil, key)
}

func (c *CByteCache) GetTo(dst []byte, key string) ([]byte, error) {
	hash := fnv.Hash32aString(key)
	shard := c.shards[hash&c.mask]
	return shard.get(dst, hash)
}
