package cbytecache

import (
	"testing"
)

func TestTypes(t *testing.T) {
	t.Run("entry copy", func(t *testing.T) {
		e := Entry{
			Key:    "test0",
			Body:   []byte("foobar"),
			Expire: 15,
		}
		cpy := e.Copy()
		assertString(t, e.Key, cpy.Key)
		assertBytes(t, e.Body, cpy.Body)
	})
}

type testq struct{}

func (q testq) Enqueue(x interface{}) error {
	e, ok := x.(Entry)
	if !ok {
		return nil
	}
	_, _, _ = e.Key, e.Body, e.Expire
	return nil
}

func BenchmarkTypes(b *testing.B) {
	b.Run("entry copy", func(b *testing.B) {
		b.ReportAllocs()
		e := Entry{
			Key:    "test0",
			Body:   []byte("foobar"),
			Expire: 15,
		}
		for i := 0; i < b.N; i++ {
			cpy := e.Copy()
			assertString(b, e.Key, cpy.Key)
			assertBytes(b, e.Body, cpy.Body)
		}
	})
	b.Run("entry enqueue", func(b *testing.B) {
		b.ReportAllocs()
		e := Entry{
			Key:    "test0",
			Body:   []byte("foobar"),
			Expire: 15,
		}
		var q testq
		for i := 0; i < b.N; i++ {
			cpy := e.Copy()
			_ = q.Enqueue(cpy)
		}
	})
}
