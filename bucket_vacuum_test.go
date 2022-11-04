package cbytecache

import (
	"testing"
	"time"

	"github.com/koykov/clock"
	"github.com/koykov/fastconv"
	"github.com/koykov/hash/fnv"
)

func TestVacuum(t *testing.T) {
	const entries = 1e6

	conf := DefaultConfig(time.Minute, &fnv.Hasher{})
	conf.Buckets = 1
	conf.Clock = clock.NewClock()
	conf.VacuumInterval = time.Minute * 2
	cache, err := New(conf)
	if err != nil {
		t.Fatal(err)
	}
	var key []byte
	for i := 0; i < entries; i++ {
		key = makeKey(key, i)
		if err = cache.Set(fastconv.B2S(key), getEntryBody(i)); err != nil {
			t.Error(err)
		}
	}
	assertSize(t, cache.Size(), CacheSize{264241152, 264222333, 18819})
	// Wait for expiration.
	conf.Clock.Jump(time.Minute)
	time.Sleep(time.Millisecond * 5)
	assertSize(t, cache.Size(), CacheSize{264241152, 0, 264241152})
	// Wait for vacuum.
	conf.Clock.Jump(time.Minute)
	time.Sleep(time.Millisecond * 5)
	assertSize(t, cache.Size(), CacheSize{1048576, 0, 1048576})
	conf.Clock.Stop()
}
