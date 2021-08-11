package cbytecache

import (
	"bytes"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/koykov/fastconv"
)

var (
	dataPool = [][]byte{
		[]byte(`{"firstName":"John","lastName":"Smith","isAlive":true,"age":27,"address":{"streetAddress":"21 2nd Street","city":"New York","state":"NY","postalCode":"10021-3100"},"phoneNumbers":[{"type":"home","number":"212 555-1234"},{"type":"office","number":"646 555-4567"},{"type":"mobile","number":"123 456-7890"}],"children":[],"spouse":null}`),
		[]byte(`{"$schema":"http://json-schema.org/schema#","title":"Product","type":"object","required":["id","name","price"],"properties":{"id":{"type":"number","description":"Product identifier"},"name":{"type":"string","description":"Name of the product"},"price":{"type":"number","minimum":0},"tags":{"type":"array","items":{"type":"string"}},"stock":{"type":"object","properties":{"warehouse":{"type":"number"},"retail":{"type":"number"}}}}}`),
		[]byte(`{"id":1,"name":"Foo","price":123,"tags":["Bar","Eek"],"stock":{"warehouse":300,"retail":20}}`),
		[]byte(`{"first name":"John","last name":"Smith","age":25,"address":{"street address":"21 2nd Street","city":"New York","state":"NY","postal code":"10021"},"phone numbers":[{"type":"home","number":"212 555-1234"},{"type":"fax","number":"646 555-4567"}],"sex":{"type":"male"}}`),
		[]byte(`{"fruit":"Apple","size":"Large","color":"Red"}`),
		[]byte(`{"quiz":{"sport":{"q1":{"question":"Which one is correct team name in NBA?","options":["New York Bulls","Los Angeles Kings","Golden State Warriros","Huston Rocket"],"answer":"Huston Rocket"}},"maths":{"q1":{"question":"5 + 7 = ?","options":["10","11","12","13"],"answer":"12"},"q2":{"question":"12 - 8 = ?","options":["1","2","3","4"],"answer":"4"}}}}`),
	}
)

func assertBytes(t testing.TB, a, b []byte) (eq bool) {
	if eq = bytes.Equal(a, b); !eq {
		t.Errorf("bytes equal fail:\nneed: %s\ngot:  %s", a, b)
	}
	return
}

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
			var buf []byte
			for i := 0; i < entries; i++ {
				buf = append(buf[:0], "key"...)
				buf = strconv.AppendInt(buf, int64(i), 10)
				if err = cache.Set(fastconv.B2S(buf), dataPool[0]); err != nil {
					t.Error(err)
				}
			}
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			var buf, dst []byte
			for i := 0; i < entries; i++ {
				buf = append(buf[:0], "key"...)
				buf = strconv.AppendInt(buf, int64(i), 10)
				if dst, err = cache.GetTo(dst[:0], fastconv.B2S(buf)); err != nil && err != ErrNotFound {
					t.Error(err)
				}
				if err == nil {
					assertBytes(t, dataPool[0], dst)
				}
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
