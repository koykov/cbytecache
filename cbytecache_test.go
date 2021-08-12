package cbytecache

import (
	"sync"
	"testing"
	"time"

	"github.com/koykov/fastconv"
)

func TestCacheIO(t *testing.T) {
	testIO := func(t *testing.T, entries int) {
		conf := DefaultConfig(time.Minute, fastconv.Fnv64aString)
		cache, err := NewCByteCache(conf)
		if err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			var key []byte
			for i := 0; i < entries; i++ {
				key = makeKey(key, i)
				if err := cache.Set(fastconv.B2S(key), getEntryBody(i)); err != nil {
					t.Error(err)
				}
			}
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			var (
				key, dst []byte
				err      error
			)
			for i := 0; i < entries; i++ {
				key = makeKey(key, i)
				if dst, err = cache.GetTo(dst[:0], fastconv.B2S(key)); err != nil {
					if err != ErrNotFound {
						t.Error(err)
					}
					continue
				}
				assertBytes(t, getEntryBody(i), dst)
			}
			wg.Done()
		}()

		wg.Wait()

		if err = cache.Close(); err != nil {
			t.Error(err)
		}
	}
	t.Run("1", func(t *testing.T) { testIO(t, 1) })
	t.Run("10", func(t *testing.T) { testIO(t, 10) })
	t.Run("100", func(t *testing.T) { testIO(t, 100) })
	t.Run("1K", func(t *testing.T) { testIO(t, 1000) })
	t.Run("10K", func(t *testing.T) { testIO(t, 10000) })
	t.Run("100K", func(t *testing.T) { testIO(t, 100000) })
	t.Run("1M", func(t *testing.T) { testIO(t, 1000000) })
}
