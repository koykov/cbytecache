package cbytecache

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"
	"time"

	"github.com/koykov/bytealg"
	"github.com/koykov/byteconv"
	"github.com/koykov/clock"
	"github.com/koykov/hash/fnv"
)

func TestDumper(t *testing.T) {
	t.Run("write", func(t *testing.T) {
		var w testDumpWriter
		conf := DefaultConfig(time.Minute, &fnv.Hasher{}, 0)
		// Writer isn't thread-safe, so use only one bucket and writer worker.
		conf.Buckets = 1
		conf.DumpWriter = &w
		conf.DumpInterval = time.Second * 30
		conf.DumpWriteWorkers = 1
		conf.Clock = clock.NewClock()
		cache, err := New(conf)
		if err != nil {
			t.Fatal(err)
		}
		var key []byte
		for i := 0; i < 10; i++ {
			key = makeKey(key, i)
			if err = cache.Set(byteconv.B2S(key), getEntryBody(i)); err != nil {
				t.Error(err)
			}
		}

		conf.Clock.Jump(time.Second * 30)
		time.Sleep(time.Millisecond * 5)

		if w.len() != 2783 {
			t.Errorf("dump size mismatch: need %d got %d", 2783, w.len())
		}
	})

	t.Run("read", func(t *testing.T) {
		var r testDumpReader
		conf := DefaultConfig(time.Hour, &fnv.Hasher{}, 0)
		// Writer isn't thread-safe, so use only one bucket and writer worker.
		conf.Buckets = 1
		conf.DumpReader = &r
		conf.DumpReadWorkers = 4
		conf.Clock = clock.NewClock()
		cache, err := New(conf)
		if err != nil {
			t.Fatal(err)
		}

		var key []byte
		for i := 0; i < 10; i++ {
			key = makeKey(key, i)
			body, err := cache.Get(byteconv.B2S(key))
			if err != nil {
				t.Error(err)
				continue
			}
			if expect := getEntryBody(i); !bytes.Equal(body, expect) {
				t.Errorf("body mismatch for key '%s':\nneed '%s'\ngot:  '%s'", string(key), string(expect), string(body))
			}
		}
	})
}

type testDumpWriter struct {
	f *os.File
	n int
}

func (w *testDumpWriter) Write(entry Entry) (n int, err error) {
	if w.f == nil {
		w.n = 0
		var fn []byte
		if fn, err = clock.Format("testdata/test--%Y-%m-%d--%H-%M-%S--%N.bin", time.Now()); err != nil {
			return
		}
		if w.f, err = os.Create(string(fn)); err != nil {
			return
		}
	}
	var buf []byte
	buf = bytealg.GrowDelta(buf, 2)
	binary.LittleEndian.PutUint16(buf, uint16(len(entry.Key)))
	buf = append(buf, entry.Key...)
	buf = bytealg.GrowDelta(buf, 4)
	binary.LittleEndian.PutUint32(buf[len(buf)-4:], uint32(len(entry.Body)))
	buf = append(buf, entry.Body...)
	buf = bytealg.GrowDelta(buf, 4)
	binary.LittleEndian.PutUint32(buf[len(buf)-4:], entry.Expire)

	n, err = w.f.Write(buf)
	w.n += n
	return
}

func (w *testDumpWriter) Flush() error {
	if w.f != nil {
		if err := w.f.Close(); err != nil {
			return err
		}
		w.f = nil
	}
	return nil
}

func (w *testDumpWriter) len() int {
	return w.n
}

type testDumpReader struct {
	f   *os.File
	buf []byte
}

func (r *testDumpReader) Read() (e Entry, err error) {
	defer func() {
		if err == io.EOF && r.f != nil {
			_ = r.f.Close()
			r.f = nil
		}
	}()

	if r.f == nil {
		if r.f, err = os.Open("testdata/example.bin"); err != nil {
			return
		}
	}
	r.buf = r.buf[:0]
	off := 0
	r.buf = bytealg.GrowDelta(r.buf, 2)
	if _, err = io.ReadAtLeast(r.f, r.buf, 2); err != nil {
		return
	}
	kl := binary.LittleEndian.Uint16(r.buf)
	off += 2
	r.buf = bytealg.GrowDelta(r.buf, int(kl))
	if _, err = io.ReadAtLeast(r.f, r.buf[off:], int(kl)); err != nil {
		return
	}
	e.Key = byteconv.B2S(r.buf[off:])
	off += int(kl)

	r.buf = bytealg.GrowDelta(r.buf, 4)
	if _, err = io.ReadAtLeast(r.f, r.buf[off:], 4); err != nil {
		return
	}
	bl := binary.LittleEndian.Uint32(r.buf[off:])
	off += 4
	r.buf = bytealg.GrowDelta(r.buf, int(bl))
	if _, err = io.ReadAtLeast(r.f, r.buf[off:], int(bl)); err != nil {
		return
	}
	e.Body = r.buf[off:]
	off += int(bl)

	r.buf = bytealg.GrowDelta(r.buf, 4)
	if _, err = io.ReadAtLeast(r.f, r.buf[off:], 4); err != nil {
		return
	}
	e.Expire = binary.LittleEndian.Uint32(r.buf[off:])
	return
}
