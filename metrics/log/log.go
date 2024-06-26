package cbytecache

import (
	"log"
	"time"
)

// LogMetrics is Log implementation of cbytecache.MetricsWriter.
//
// Don't use in production. Only for debug purposes.
type LogMetrics struct {
	key string
}

func NewLogMetrics(key string) *LogMetrics {
	m := &LogMetrics{key}
	return m
}

func (m LogMetrics) Alloc(bucket string, size uint32) {
	log.Printf("cbytecache %s: alloc new arena with size %d in bucket %s\n", m.key, size, bucket)
}

func (m LogMetrics) Fill(bucket string, size uint32) {
	log.Printf("cbytecache %s: fill arena with size %d bytes of bucket %s\n", m.key, size, bucket)
}

func (m LogMetrics) Reset(bucket string, size uint32) {
	log.Printf("cbytecache %s: reset arena with size %d of bucket %s\n", m.key, size, bucket)
}

func (m LogMetrics) Release(bucket string, size uint32) {
	log.Printf("cbytecache %s: release arena with size %d bytes of bucket %s\n", m.key, size, bucket)
}

func (m LogMetrics) Set(bucket string, dur time.Duration) {
	log.Printf("cbytecache %s: set new entry to bucket %s took %s\n", m.key, bucket, dur)
}

func (m LogMetrics) Del(bucket string) {
	log.Printf("cbytecache %s: delete entry from bucket %s\n", m.key, bucket)
}

func (m LogMetrics) Evict(bucket string, _ bool) {
	log.Printf("cbytecache %s: evict entry from bucket %s\n", m.key, bucket)
}

func (m LogMetrics) Miss(bucket string) {
	log.Printf("cbytecache %s: cache miss in bucket %s\n", m.key, bucket)
}

func (m LogMetrics) Hit(bucket string, dur time.Duration) {
	log.Printf("cbytecache %s: cache hit in bucket %s took %s\n", m.key, bucket, dur)
}

func (m LogMetrics) Expire(bucket string) {
	log.Printf("cbytecache %s: hit expired entry in bucket %s\n", m.key, bucket)
}

func (m LogMetrics) Corrupt(bucket string) {
	log.Printf("cbytecache %s: hit corrupted entry in bucket %s\n", m.key, bucket)
}

func (m LogMetrics) Collision(bucket string) {
	log.Printf("cbytecache %s: keys collision in bucket %s\n", m.key, bucket)
}

func (m LogMetrics) NoSpace(bucket string) {
	log.Printf("cbytecache %s: no space in bucket %s\n", m.key, bucket)
}

func (m LogMetrics) Dump(bucket string) {
	log.Printf("cbytecache %s: dump entry of bucket #%s\n", m.key, bucket)
}

func (m LogMetrics) Load(bucket string) {
	log.Printf("cbytecache %s: load dumped entry to bucket #%s\n", m.key, bucket)
}

var _ = NewLogMetrics
