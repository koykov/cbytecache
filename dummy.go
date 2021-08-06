package cbytecache

type DummyMetrics struct{}

func (*DummyMetrics) Alloc(_ uint32)   {}
func (*DummyMetrics) Free(_ uint32)    {}
func (*DummyMetrics) Release(_ uint32) {}
func (*DummyMetrics) Set(_ uint16)     {}
func (*DummyMetrics) Evict(_ uint16)   {}
func (*DummyMetrics) Miss()            {}
func (*DummyMetrics) Hit()             {}
func (*DummyMetrics) Expire()          {}
func (*DummyMetrics) Corrupt()         {}
func (*DummyMetrics) NoSpace()         {}

type DummyLog struct{}

func (*DummyLog) Printf(string, ...interface{}) {}
func (*DummyLog) Print(...interface{})          {}
func (*DummyLog) Println(...interface{})        {}
