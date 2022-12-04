package cbytecache

import (
	"errors"
	"testing"
	"time"

	"github.com/koykov/clock"
	"github.com/koykov/fastconv"
	"github.com/koykov/hash/fnv"
)

func TestExpire(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		conf := DefaultConfig(time.Minute, &fnv.Hasher{})
		conf.Clock = clock.NewClock()
		cache, err := New(conf)
		if err != nil {
			t.Fatal(err)
		}
		if err = cache.Set("foo", getEntryBody(0)); err != nil {
			t.Fatal(err)
		}
		// Wait for expiration.
		conf.Clock.Jump(time.Minute + time.Second)
		time.Sleep(time.Millisecond * 5)
		if _, err = cache.Get("foo"); err == nil {
			err = errors.New(`entry got from the cache but expects "not found" error`)
		}
		if err != ErrNotFound {
			t.Errorf("get expired entry failed with error: %s", err.Error())
		}
		if err = cache.Close(); err != nil {
			t.Error(err.Error())
		}
	})
	t.Run("multi", func(t *testing.T) {
		const (
			entries    = 1e5
			fullSize   = 26322333
			expectSize = 0
		)

		conf := DefaultConfig(time.Minute, &fnv.Hasher{})
		conf.Clock = clock.NewClock()
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
		if size := cache.Size(); size.Used() != fullSize {
			t.Error("wrong cache size after set: need", fullSize, "got", size)
		}
		// Wait for expiration.
		conf.Clock.Jump(time.Minute + time.Second)
		time.Sleep(time.Millisecond * 5)
		conf.Clock.Stop()
		if size := cache.Size(); size.Used() != expectSize {
			t.Error("wrong cache size after expire: need", expectSize, "got", size)
		}
	})
}
