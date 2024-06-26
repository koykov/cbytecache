package cbytecache

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	cacheTotal       = "total"
	cacheUsed        = "used"
	cacheFree        = "free"
	cacheEntryTotal  = "entry_total"
	cacheEntryDelete = "entry_delete"

	cacheIOSet       = "set"
	cacheIOEvict     = "evict"
	cacheIOMiss      = "miss"
	cacheIOHit       = "hit"
	cacheIODel       = "del"
	cacheIOExpire    = "expire"
	cacheIOCorrupt   = "corrupt"
	cacheIOCollision = "collision"
	cacheIONoSpace   = "no space"

	speedWrite = "write"
	speedRead  = "read"

	arenaTotal = "total"
	arenaUsed  = "used"
	arenaFree  = "free"

	arenaIOAlloc   = "alloc"
	arenaIORelease = "release"
	arenaIOReset   = "reset"
	arenaIOFill    = "fill"

	dumpIODump = "dump"
	dumpIOLoad = "load"
)

// PrometheusMetrics is a Prometheus implementation of cbytecache.MetricsWriter.
type PrometheusMetrics struct {
	key  string
	prec time.Duration
}

var (
	promSize, promArena             *prometheus.GaugeVec
	promIO, promArenaIO, promDumpIO *prometheus.CounterVec
	promSpeed                       *prometheus.HistogramVec

	_ = NewPrometheusMetrics
)

func init() {
	promSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cbytecache_size",
		Help: "Total, used and free cache (bucket) size in bytes.",
	}, []string{"cache", "bucket", "type"})
	promIO = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cbytecache_io",
		Help: "Count cache IO operations calls.",
	}, []string{"cache", "bucket", "op"})

	promArena = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cbytecache_arena",
		Help: "Arenas count in cache (bucket).",
	}, []string{"cache", "bucket", "type"})
	promArenaIO = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cbytecache_arena_io",
		Help: "Count arena IO operations calls.",
	}, []string{"cache", "bucket", "op"})

	promDumpIO = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cbytecache_dump",
		Help: "Count dump IO operations calls.",
	}, []string{"cache", "bucket", "op"})

	speedBuckets := append(prometheus.DefBuckets, []float64{15, 20, 30, 40, 50, 100, 150, 200, 250, 500, 1000, 1500, 2000, 3000, 5000}...)
	promSpeed = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cbytecache_io_speed",
		Help:    "Cache IO operations speed.",
		Buckets: speedBuckets,
	}, []string{"cache", "bucket", "op"})

	prometheus.MustRegister(promSize, promIO, promDumpIO, promArena, promArenaIO, promSpeed)
}

func NewPrometheusMetrics(key string) *PrometheusMetrics {
	return NewPrometheusMetricsWP(key, time.Nanosecond)
}

func NewPrometheusMetricsWP(key string, precision time.Duration) *PrometheusMetrics {
	if precision == 0 {
		precision = time.Nanosecond
	}
	m := &PrometheusMetrics{
		key:  key,
		prec: precision,
	}
	return m
}

func (m PrometheusMetrics) Alloc(bucket string, size uint32) {
	promSize.WithLabelValues(m.key, bucket, cacheTotal).Add(float64(size))
	promSize.WithLabelValues(m.key, bucket, cacheFree).Add(float64(size))

	promArena.WithLabelValues(m.key, bucket, arenaTotal).Inc()
	promArena.WithLabelValues(m.key, bucket, arenaFree).Inc()
	promArenaIO.WithLabelValues(m.key, bucket, arenaIOAlloc).Inc()
}

func (m PrometheusMetrics) Fill(bucket string, size uint32) {
	promSize.WithLabelValues(m.key, bucket, cacheUsed).Add(float64(size))
	promSize.WithLabelValues(m.key, bucket, cacheFree).Add(-float64(size))

	promArena.WithLabelValues(m.key, bucket, arenaUsed).Inc()
	promArena.WithLabelValues(m.key, bucket, arenaFree).Dec()
	promArenaIO.WithLabelValues(m.key, bucket, arenaIOFill).Inc()
}

func (m PrometheusMetrics) Reset(bucket string, size uint32) {
	promSize.WithLabelValues(m.key, bucket, cacheUsed).Add(-float64(size))
	promSize.WithLabelValues(m.key, bucket, cacheFree).Add(float64(size))

	promArena.WithLabelValues(m.key, bucket, arenaUsed).Dec()
	promArena.WithLabelValues(m.key, bucket, arenaFree).Inc()
	promArenaIO.WithLabelValues(m.key, bucket, arenaIOReset).Inc()
}

func (m PrometheusMetrics) Release(bucket string, size uint32) {
	promSize.WithLabelValues(m.key, bucket, cacheTotal).Add(-float64(size))
	promSize.WithLabelValues(m.key, bucket, cacheFree).Add(-float64(size))

	promArena.WithLabelValues(m.key, bucket, arenaTotal).Dec()
	promArena.WithLabelValues(m.key, bucket, arenaFree).Dec()
	promArenaIO.WithLabelValues(m.key, bucket, arenaIORelease).Inc()
}

func (m PrometheusMetrics) Set(bucket string, dur time.Duration) {
	promSize.WithLabelValues(m.key, bucket, cacheEntryTotal).Inc()
	promIO.WithLabelValues(m.key, bucket, cacheIOSet).Inc()
	promSpeed.WithLabelValues(m.key, bucket, speedWrite).Observe(float64(dur.Nanoseconds() / int64(m.prec)))
}

func (m PrometheusMetrics) Del(bucket string) {
	promSize.WithLabelValues(m.key, bucket, cacheEntryDelete).Inc()
	promIO.WithLabelValues(m.key, bucket, cacheIODel).Inc()
}

func (m PrometheusMetrics) Evict(bucket string, alive bool) {
	promSize.WithLabelValues(m.key, bucket, cacheEntryTotal).Dec()
	if !alive {
		promSize.WithLabelValues(m.key, bucket, cacheEntryDelete).Dec()
	}
	promIO.WithLabelValues(m.key, bucket, cacheIOEvict).Inc()
}

func (m PrometheusMetrics) Miss(bucket string) {
	promIO.WithLabelValues(m.key, bucket, cacheIOMiss).Inc()
}

func (m PrometheusMetrics) Hit(bucket string, dur time.Duration) {
	promIO.WithLabelValues(m.key, bucket, cacheIOHit).Inc()
	promSpeed.WithLabelValues(m.key, bucket, speedRead).Observe(float64(dur.Nanoseconds() / int64(m.prec)))
}

func (m PrometheusMetrics) Expire(bucket string) {
	promIO.WithLabelValues(m.key, bucket, cacheIOExpire).Inc()
}

func (m PrometheusMetrics) Corrupt(bucket string) {
	promIO.WithLabelValues(m.key, bucket, cacheIOCorrupt).Inc()
}

func (m PrometheusMetrics) Collision(bucket string) {
	promIO.WithLabelValues(m.key, bucket, cacheIOCollision).Inc()
}

func (m PrometheusMetrics) NoSpace(bucket string) {
	promIO.WithLabelValues(m.key, bucket, cacheIONoSpace).Inc()
}

func (m PrometheusMetrics) Dump(bucket string) {
	promDumpIO.WithLabelValues(m.key, bucket, dumpIODump).Inc()
}

func (m PrometheusMetrics) Load(bucket string) {
	promDumpIO.WithLabelValues(m.key, bucket, dumpIOLoad).Inc()
}
