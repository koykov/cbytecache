package cbytecache

import (
	"bytes"
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

func assertBytes(t testing.TB, a, b []byte) {
	if !bytes.Equal(a, b) {
		t.Errorf("bytes equal fail:\nneed: %s\ngot:  %s", a, b)
	}
}

func TestCacheIOSingle(t *testing.T) {
	conf := DefaultConfig(time.Minute, fastconv.Fnv64aString)
	cache, err := NewCByteCache(conf)
	if err != nil {
		t.Fatal(err)
	}
	if err = cache.Set("key135", dataPool[0]); err != nil {
		t.Error(err)
	}
	var dst []byte
	if dst, err = cache.Get("key135"); err != nil {
		t.Error(err)
	}
	assertBytes(t, dataPool[0], dst)
}

// import (
// 	"math/rand"
// 	"strings"
// 	"testing"
// 	"time"
// )
//
// var (
// 	dataPool = [][]byte{
// 		[]byte(`{"firstName":"John","lastName":"Smith","isAlive":true,"age":27,"address":{"streetAddress":"21 2nd Street","city":"New York","state":"NY","postalCode":"10021-3100"},"phoneNumbers":[{"type":"home","number":"212 555-1234"},{"type":"office","number":"646 555-4567"},{"type":"mobile","number":"123 456-7890"}],"children":[],"spouse":null}`),
// 		[]byte(`{"$schema":"http://json-schema.org/schema#","title":"Product","type":"object","required":["id","name","price"],"properties":{"id":{"type":"number","description":"Product identifier"},"name":{"type":"string","description":"Name of the product"},"price":{"type":"number","minimum":0},"tags":{"type":"array","items":{"type":"string"}},"stock":{"type":"object","properties":{"warehouse":{"type":"number"},"retail":{"type":"number"}}}}}`),
// 		[]byte(`{"id":1,"name":"Foo","price":123,"tags":["Bar","Eek"],"stock":{"warehouse":300,"retail":20}}`),
// 		[]byte(`{"first name":"John","last name":"Smith","age":25,"address":{"street address":"21 2nd Street","city":"New York","state":"NY","postal code":"10021"},"phone numbers":[{"type":"home","number":"212 555-1234"},{"type":"fax","number":"646 555-4567"}],"sex":{"type":"male"}}`),
// 		[]byte(`{"fruit":"Apple","size":"Large","color":"Red"}`),
// 		[]byte(`{"quiz":{"sport":{"q1":{"question":"Which one is correct team name in NBA?","options":["New York Bulls","Los Angeles Kings","Golden State Warriros","Huston Rocket"],"answer":"Huston Rocket"}},"maths":{"q1":{"question":"5 + 7 = ?","options":["10","11","12","13"],"answer":"12"},"q2":{"question":"12 - 8 = ?","options":["1","2","3","4"],"answer":"4"}}}}`),
// 	}
// 	chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
// 	keys  = fillKeys()
// 	key   strings.Builder
// )
//
// func fillKeys() []string {
// 	keys := make([]string, 10000000)
// 	for i := 0; i < 10000000; i++ {
// 		keys[i] = randKey(20)
// 	}
// 	return keys
// }
//
// func randKey(n int) string {
// 	rand.Seed(time.Now().UnixNano())
// 	key.Reset()
// 	for i := 0; i < n; i++ {
// 		key.WriteRune(chars[rand.Intn(len(chars))])
// 	}
// 	return key.String()
// }
//
// func cbcBench(b *testing.B, alg uint) {
// 	// conf := Config{Buckets: 16}
// 	// c := NewCByteCache(&conf)
// 	// c.Alg = alg
// 	// b.ResetTimer()
// 	// b.ReportAllocs()
// 	// ki := 0
// 	// var key string
// 	// for i := 0; i < b.N; i++ {
// 	// 	//key := randKey(20)
// 	// 	if i < 10000000 {
// 	// 		key = keys[ki]
// 	// 		ki++
// 	// 	} else {
// 	// 		ki = 0
// 	// 		key = keys[ki]
// 	// 	}
// 	//
// 	// 	data := dataPool[rand.Intn(len(dataPool))]
// 	// 	_ = c.Set(key, data)
// 	// 	//_, _ = c.Get(key)
// 	// 	//r, _ := c.Get(key)
// 	// 	//if !bytes.Equal(r, data) {
// 	// 	//	b.Error(r, data)
// 	// 	//}
// 	// }
// }
//
// func BenchmarkCbcHashMap(b *testing.B) {
// 	// cbcBench(b, AlgHashMap)
// }
//
// func BenchmarkCbcSlicePair(b *testing.B) {
// 	// cbcBench(b, AlgSlicePair)
// }
//
// func BenchmarkCbcSlicePairLr(b *testing.B) {
// 	// cbcBench(b, AlgSlicePairLr)
// }
//
// func BenchmarkCbcHashEntryMap(b *testing.B) {
// 	// cbcBench(b, AlgHashEntryMap)
// }
