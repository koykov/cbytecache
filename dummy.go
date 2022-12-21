package cbytecache

import "time"

type DummyMetrics struct{}

func (DummyMetrics) Alloc(_ string, _ uint32)      {}
func (DummyMetrics) Fill(_ string, _ uint32)       {}
func (DummyMetrics) Reset(_ string, _ uint32)      {}
func (DummyMetrics) Release(_ string, _ uint32)    {}
func (DummyMetrics) Set(_ string, _ time.Duration) {}
func (DummyMetrics) Hit(_ string, _ time.Duration) {}
func (DummyMetrics) Del(_ string)                  {}
func (DummyMetrics) Evict(_ string)                {}
func (DummyMetrics) Miss(_ string)                 {}
func (DummyMetrics) Expire(_ string)               {}
func (DummyMetrics) Corrupt(_ string)              {}
func (DummyMetrics) Collision(_ string)            {}
func (DummyMetrics) NoSpace(_ string)              {}
func (DummyMetrics) Dump(_ string)                 {}
func (DummyMetrics) Load(_ string)                 {}

var dummyMetrics = DummyMetrics{}
