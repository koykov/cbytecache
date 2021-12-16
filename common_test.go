package cbytecache

import (
	"bytes"
	"strconv"
	"testing"
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
	dpLen = len(dataPool)
)

func makeKey(dst []byte, i int) []byte {
	dst = append(dst[:0], "key"...)
	dst = strconv.AppendInt(dst, int64(i), 10)
	return dst
}

func getEntryBody(i int) []byte {
	return dataPool[i%dpLen]
}

func assertBytes(t testing.TB, a, b []byte) (eq bool) {
	if eq = bytes.Equal(a, b); !eq {
		t.Errorf("bytes equal fail:\nneed: %s\ngot:  %s", a, b)
	}
	return
}

func assertSize(t testing.TB, a, b CacheSize) {
	if !a.Equal(b) {
		t.Errorf("cache size equal fail:\nneed: %s\ngot:  %s", a, b)
	}
}
