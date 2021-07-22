package cbytecache

type MetricsWriter interface {
	Set(len int)
	Miss()
	HitOK()
	HitExpired()
	HitCorrupted()
	Evict(len int)
}
