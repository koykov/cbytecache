package cbytecache

import "github.com/koykov/fastconv"

type CByteCache struct {
	config *Config
	shards []*shard
	mask   uint64
	Alg    uint
}

func NewCByteCache(config *Config) *CByteCache {
	shards := make([]*shard, config.Shards)
	for i := range shards {
		shards[i] = newShard()
	}

	c := &CByteCache{
		config: config,
		shards: shards,
		mask:   uint64(config.Shards - 1),
	}
	return c
}

func (c *CByteCache) Set(key string, data []byte) error {
	hash := fastconv.Fnv64aString(key)
	shard := c.shards[hash&c.mask]
	return shard.set(hash, data)
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	hash := fastconv.Fnv64aString(key)
	shard := c.shards[hash&c.mask]
	return shard.get(hash)
}
