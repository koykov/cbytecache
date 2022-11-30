package cbytecache

import "time"

type DummyMetrics struct{}

func (DummyMetrics) Alloc(_ string)                       {}
func (DummyMetrics) Fill(_ string)                        {}
func (DummyMetrics) Reset(_ string)                       {}
func (DummyMetrics) Release(_ string)                     {}
func (DummyMetrics) ArenaMap(_ string, _, _, _, _ uint32) {}
func (DummyMetrics) Set(_ string, _ time.Duration)        {}
func (DummyMetrics) Hit(_ string, _ time.Duration)        {}
func (DummyMetrics) Evict(_ string)                       {}
func (DummyMetrics) Miss(_ string)                        {}
func (DummyMetrics) Expire(_ string)                      {}
func (DummyMetrics) Corrupt(_ string)                     {}
func (DummyMetrics) Collision(_ string)                   {}
func (DummyMetrics) NoSpace(_ string)                     {}
func (DummyMetrics) Dump()                                {}
func (DummyMetrics) Load()                                {}

var dummyMetrics = DummyMetrics{}
