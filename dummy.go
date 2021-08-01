package cbytecache

type DummyMetrics struct{}

func (*DummyMetrics) Grow(_ uint32)   {}
func (*DummyMetrics) Reduce(_ uint32) {}
func (*DummyMetrics) Set(_ uint16)    {}
func (*DummyMetrics) Miss()           {}
func (*DummyMetrics) Hit()            {}
func (*DummyMetrics) Expire()         {}
func (*DummyMetrics) Corrupt()        {}
func (*DummyMetrics) Evict(_ uint16)  {}

type DummyLog struct{}

func (*DummyLog) Printf(string, ...interface{}) {}
func (*DummyLog) Print(...interface{})          {}
func (*DummyLog) Println(...interface{})        {}
