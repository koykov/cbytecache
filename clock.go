package cbytecache

import "time"

type Clock interface {
	Now() time.Time
	Jump(delta time.Duration)
}

type nativeClock struct{}

func (n nativeClock) Now() time.Time {
	return time.Now()
}

func (n nativeClock) Jump(_ time.Duration) {}
