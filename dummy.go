package cbytecache

import "time"

type DummyMetrics struct{}

func (DummyMetrics) Alloc(_ string, _ uint32)                {}
func (DummyMetrics) Release(_ string, _ uint32)              {}
func (DummyMetrics) Set(_ string, _ uint32, _ time.Duration) {}
func (DummyMetrics) Reset(_ string, _ int)                   {}
func (DummyMetrics) Evict(_ string, _ uint32)                {}
func (DummyMetrics) Miss(_ string)                           {}
func (DummyMetrics) Hit(_ string, _ time.Duration)           {}
func (DummyMetrics) Expire(_ string)                         {}
func (DummyMetrics) Corrupt(_ string)                        {}
func (DummyMetrics) Collision(_ string)                      {}
func (DummyMetrics) NoSpace(_ string)                        {}
func (DummyMetrics) Dump()                                   {}
func (DummyMetrics) Load()                                   {}

var dummyMetrics = DummyMetrics{}
