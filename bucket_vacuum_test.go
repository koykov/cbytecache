package cbytecache

import (
	"testing"
	"time"

	"github.com/koykov/byteconv"
	"github.com/koykov/clock"
	"github.com/koykov/hash/fnv"
)

func TestVacuum(t *testing.T) {
	const entries = 1e6

	conf := DefaultConfig(time.Minute, &fnv.Hasher{}, 0)
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
		if err = cache.Set(byteconv.B2S(key), getEntryBody(i)); err != nil {
			t.Error(err)
		}
	}
	assertSize(t, cache.Size(), CacheSize{264224768, 264222333, 2435})
	// Wait for expiration.
	conf.Clock.Jump(time.Minute + time.Second)
	time.Sleep(time.Millisecond * 5)
	assertSize(t, cache.Size(), CacheSize{264224768, 0, 264224768})
	// Wait for vacuum.
	conf.Clock.Jump(time.Minute)
	time.Sleep(time.Millisecond * 5)
	assertSize(t, cache.Size(), CacheSize{132120576, 0, 132120576})
	conf.Clock.Stop()
}
