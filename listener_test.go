package cbytecache

import (
	"bytes"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/koykov/clock"
	"github.com/koykov/hash/fnv"
)

type countListener struct {
	k, n, b, c uint32
}

func (l *countListener) Listen(key string, body []byte) error {
	if len(key) < 3 || key[:3] != "key" {
		atomic.AddUint32(&l.k, 1)
		return fmt.Errorf("corrupted key: %s", key)
	}
	raw := key[3:]
	n, err := strconv.Atoi(raw)
	if err != nil {
		atomic.AddUint32(&l.n, 1)
		return err
	}
	expect := getEntryBody(n)
	if !bytes.Equal(expect, body) {
		atomic.AddUint32(&l.b, 1)
		return fmt.Errorf("mismatch body and expactation")
	}
	atomic.AddUint32(&l.c, 1)
	return nil
}

func (l countListener) Close() error { return nil }

func (l *countListener) stats() (uint32, uint32, uint32, uint32) {
	return atomic.LoadUint32(&l.k), atomic.LoadUint32(&l.n), atomic.LoadUint32(&l.b), atomic.LoadUint32(&l.c)
}

func TestListener(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		const count = 1000

		var l countListener

		conf := DefaultConfig(time.Minute, &fnv.Hasher{})
		conf.Clock = clock.NewClock()
		conf.ExpireListener = &l
		cache, err := New(conf)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < count; i++ {
			key := fmt.Sprintf("key%d", i)
			if err = cache.Set(key, getEntryBody(i)); err != nil {
				t.Fatal(err)
			}
		}
		// Wait for expiration.
		conf.Clock.Jump(time.Minute)
		time.Sleep(time.Millisecond * 5)

		// Check counters.
		k, n, b, c := l.stats()
		if k != 0 || n != 0 || b != 0 || c != count {
			t.Errorf("unexpected stats: %d, %d, %d, %d", k, n, b, c)
		}

		// Close cache.
		if err = cache.Close(); err != nil {
			t.Error(err.Error())
		}
	})
}
