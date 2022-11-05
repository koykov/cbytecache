package cbytecache

type DummyMetrics struct{}

func (DummyMetrics) Alloc(_ uint32)   {}
func (DummyMetrics) Free(_ uint32)    {}
func (DummyMetrics) Release(_ uint32) {}
func (DummyMetrics) Set(_ uint32)     {}
func (DummyMetrics) Evict()           {}
func (DummyMetrics) Miss()            {}
func (DummyMetrics) Hit()             {}
func (DummyMetrics) Expire()          {}
func (DummyMetrics) Corrupt()         {}
func (DummyMetrics) Collision()       {}
func (DummyMetrics) NoSpace()         {}
func (DummyMetrics) Dump()            {}
func (DummyMetrics) Load()            {}

var dummyMetrics = DummyMetrics{}
