package cbytecache

import (
	"encoding/binary"
	"os"
	"testing"
	"time"

	"github.com/koykov/bytealg"
	"github.com/koykov/clock"
	"github.com/koykov/fastconv"
	"github.com/koykov/hash/fnv"
)

func TestDump(t *testing.T) {
	t.Run("write", func(t *testing.T) {
		var w testDumpWriter
		conf := DefaultConfig(time.Minute, &fnv.Hasher{})
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
			if err = cache.Set(fastconv.B2S(key), getEntryBody(i)); err != nil {
				t.Error(err)
			}
		}

		conf.Clock.Jump(time.Second * 30)
		time.Sleep(time.Millisecond * 5)

		if w.len() != 2783 {
			t.Errorf("dump size mismatch: need %d got %d", 2783, w.len())
		}
	})
}

type testDumpWriter struct {
	f *os.File
	n int
}

func (w *testDumpWriter) Write(entry Entry) (err error) {
	if w.f == nil {
		w.n = 0
		var fn []byte
		if fn, err = clock.Format("testdata/test--%Y-%m-%d--%H-%M-%S--%N.bin", time.Now()); err != nil {
			return err
		}
		if w.f, err = os.Create(string(fn)); err != nil {
			return err
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

	var n int
	n, err = w.f.Write(buf)
	w.n += n
	return
}

func (w *testDumpWriter) Close() error {
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
