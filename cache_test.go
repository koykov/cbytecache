package cbytecache

import (
	"bytes"
	"testing"
	"time"

	"github.com/koykov/byteconv"
	"github.com/koykov/hash/fnv"
)

func TestIO(t *testing.T) {
	testIO := func(t *testing.T, entries int, verbose bool) {
		conf := DefaultConfig(time.Minute, &fnv.Hasher{}, 0)
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
			if err := cache.Set(byteconv.B2S(key), getEntryBody(i)); err != nil {
				wf++
				t.Error(err)
			}
		}

		for i := 0; i < entries; i++ {
			r++
			key = makeKey(key, i)
			if dst, err = cache.GetTo(dst[:0], byteconv.B2S(key)); err != nil {
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

func TestDelete(t *testing.T) {
	conf := DefaultConfig(time.Minute, &fnv.Hasher{}, 0)
	cache, err := New(conf)
	if err != nil {
		t.Fatal(err)
	}
	if err = cache.Set("foobar", getEntryBody(0)); err != nil {
		t.Fatal(err)
	}
	if err = cache.Delete("foobar"); err != nil {
		t.Fatal(err)
	}
	if _, err = cache.Get("foobar"); err != ErrNotFound {
		t.Errorf("error mismatch: need '%s', got '%s'", ErrNotFound.Error(), err.Error())
	}
}

func TestExtract(t *testing.T) {
	conf := DefaultConfig(time.Minute, &fnv.Hasher{}, 0)
	cache, err := New(conf)
	if err != nil {
		t.Fatal(err)
	}
	body := getEntryBody(0)
	if err = cache.Set("foobar", body); err != nil {
		t.Fatal(err)
	}
	var b []byte
	if b, err = cache.Extract("foobar"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, body) {
		t.Errorf("entry mismatch: need '%s', got '%s'", string(body), string(b))
	}
	if _, err = cache.Get("foobar"); err != ErrNotFound {
		t.Errorf("error mismatch: need '%s', got '%s'", ErrNotFound.Error(), err.Error())
	}
}
