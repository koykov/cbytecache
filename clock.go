package cbytecache

import (
	"context"
	"time"
)

type Clock interface {
	Start()
	Stop()
	Active() bool
	Now() time.Time
	Jump(delta time.Duration)
	Schedule(d time.Duration, fn func())
}

type NativeClock struct {
	cancel []context.CancelFunc
}

func (n NativeClock) Start() {}

func (n NativeClock) Stop() {
	if l := len(n.cancel); l > 0 {
		for i := 0; i < l; i++ {
			n.cancel[i]()
		}
	}
}

func (n NativeClock) Active() bool { return true }

func (n NativeClock) Now() time.Time {
	return time.Now()
}

func (n NativeClock) Jump(_ time.Duration) {}

func (n *NativeClock) Schedule(d time.Duration, fn func()) {
	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = append(n.cancel, cancel)
	go func(ctx context.Context) {
		t := time.NewTicker(d)
		for {
			select {
			case <-t.C:
				fn()
			case <-ctx.Done():
				t.Stop()
				return
			}
		}
	}(ctx)
}
