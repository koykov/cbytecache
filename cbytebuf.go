package cbytecache

import "github.com/koykov/fastconv"

type CByteCache struct {
	config *Config
	shards []*shard
	mask   uint64
	Alg    uint
}

const (
	AlgHashMap   = 0
	AlgSlicePair = 1
)

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
	switch c.Alg {
	case 0:
		return shard.set0(hash, data)
	case 1:
		return shard.set1(hash, data)
	}
	return nil
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	hash := fastconv.Fnv64aString(key)
	shard := c.shards[hash&c.mask]
	switch c.Alg {
	case 0:
		return shard.get0(hash)
	case 1:
		return shard.get1(hash)
	}
	return nil, nil
}
