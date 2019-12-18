package cbytecache

import "github.com/koykov/fastconv"

type CByteCache struct {
	config *Config
	shards []*shard
	mask   uint64
	Alg    uint
}

const (
	AlgHashMap     = 0
	AlgSlicePair   = 1
	AlgSlicePairLr = 2
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
	case AlgHashMap:
		return shard.set0(hash, data)
	case AlgSlicePair:
		return shard.set1(hash, data)
	case AlgSlicePairLr:
		return shard.set2(hash, data)
	}
	return nil
}

func (c *CByteCache) Get(key string) ([]byte, error) {
	hash := fastconv.Fnv64aString(key)
	shard := c.shards[hash&c.mask]
	switch c.Alg {
	case AlgHashMap:
		return shard.get0(hash)
	case AlgSlicePair:
		return shard.get1(hash)
	case AlgSlicePairLr:
		return shard.get2(hash)

	}
	return nil, nil
}
