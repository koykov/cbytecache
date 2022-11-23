package cbytecache

import "time"

type DummyMetrics struct{}

func (DummyMetrics) Alloc(_ string, _ uint32)                {}
func (DummyMetrics) Release(_ string, _ uint32)              {}
func (DummyMetrics) Set(_ string, _ uint32, _ time.Duration) {}
func (DummyMetrics) ArenaAlloc(_ string, _ bool)             {}
func (DummyMetrics) ArenaReset(_ string)                     {}
func (DummyMetrics) ArenaFill(_ string)                      {}
func (DummyMetrics) ArenaRelease(_ string)                   {}
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
