package cbytecache

type DummyMetrics struct{}

func (*DummyMetrics) Alloc(_ string, _ uint32)   {}
func (*DummyMetrics) Free(_ string, _ uint32)    {}
func (*DummyMetrics) Release(_ string, _ uint32) {}
func (*DummyMetrics) Set(_ string, _ uint32)     {}
func (*DummyMetrics) Evict(_ string, _ uint32)   {}
func (*DummyMetrics) Miss(_ string)              {}
func (*DummyMetrics) Hit(_ string)               {}
func (*DummyMetrics) Expire(_ string)            {}
func (*DummyMetrics) Corrupt(_ string)           {}
func (*DummyMetrics) Collision(_ string)         {}
func (*DummyMetrics) NoSpace(_ string)           {}

type DummyLog struct{}

func (*DummyLog) Printf(string, ...interface{}) {}
func (*DummyLog) Print(...interface{})          {}
func (*DummyLog) Println(...interface{})        {}

var (
	dummyMetrics = &DummyMetrics{}
	dummyLog     = &DummyLog{}
)
