package cbytecache

type DummyMetrics struct{}

func (*DummyMetrics) Set(_ int)     {}
func (*DummyMetrics) Miss()         {}
func (*DummyMetrics) HitOK()        {}
func (*DummyMetrics) HitExpired()   {}
func (*DummyMetrics) HitCorrupted() {}
func (*DummyMetrics) Evict(_ int)   {}

type DummyLog struct{}

func (*DummyLog) Printf(string, ...interface{}) {}
func (*DummyLog) Print(...interface{})          {}
func (*DummyLog) Println(...interface{})        {}
