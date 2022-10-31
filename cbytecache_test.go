package cbytecache

import (
	"testing"
	"time"

	"github.com/koykov/clock"
	"github.com/koykov/fastconv"
	"github.com/koykov/hash/fnv"
)

func TestCacheIO(t *testing.T) {
	testIO := func(t *testing.T, entries int, verbose bool) {
		conf := DefaultConfig(time.Minute, &fnv.Hasher{})
		cache, err := New(conf)
		if err != nil {
			t.Fatal(err)
		}

		var (
			key, dst           []byte
			w, wf, r, rf, r404 int
		)

		for i := 0; i < entries; i++ {
			w++
			key = makeKey(key, i)
			if err := cache.Set(fastconv.B2S(key), getEntryBody(i)); err != nil {
				wf++
				t.Error(err)
			}
		}

		for i := 0; i < entries; i++ {
			r++
			key = makeKey(key, i)
			if dst, err = cache.GetTo(dst[:0], fastconv.B2S(key)); err != nil {
				rf++
				r404++
				if err != ErrNotFound {
					r404--
					t.Error(err)
				}
				continue
			}
			assertBytes(t, getEntryBody(i), dst)
		}

		if verbose {
			t.Logf("write: %d\nwrite fail: %d\nread: %d\nread fail: %d\nread 404: %d", w, wf, r, rf, r404)
		}

		if err = cache.Close(); err != nil {
			t.Error(err)
		}
	}

	verbose := false
	t.Run("1", func(t *testing.T) { testIO(t, 1, verbose) })
	t.Run("10", func(t *testing.T) { testIO(t, 10, verbose) })
	t.Run("100", func(t *testing.T) { testIO(t, 100, verbose) })
	t.Run("1K", func(t *testing.T) { testIO(t, 1000, verbose) })
	t.Run("10K", func(t *testing.T) { testIO(t, 10000, verbose) })
	t.Run("100K", func(t *testing.T) { testIO(t, 100000, verbose) })
	t.Run("1M", func(t *testing.T) { testIO(t, 1000000, verbose) })
}

func TestCByteCacheVacuum(t *testing.T) {
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
