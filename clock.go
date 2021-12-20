package cbytecache

import (
	"context"
	"sync"
	"time"
)

// Clock describes timer helper with testing features (like Jump).
type Clock interface {
	// Start the clock.
	Start()
	// Stop the clock.
	Stop()
	// Active checks if close is started.
	Active() bool
	// Now returns current time considering jumps.
	Now() time.Time
	// Jump performs time travel to delta (maybe negative). Now and scheduled jobs considers jumps.
	Jump(delta time.Duration)
	// Schedule registers fn to call at every d.
	Schedule(d time.Duration, fn func())
}

// NativeClock is a primitive clock based on time package.
//
// Jump doesn't work in that implementation.
type NativeClock struct {
	mux    sync.Mutex
	cancel []context.CancelFunc
}

func (n *NativeClock) Start() {}

func (n *NativeClock) Stop() {
	n.mux.Lock()
	if l := len(n.cancel); l > 0 {
		for i := 0; i < l; i++ {
			n.cancel[i]()
		}
	}
	n.cancel = n.cancel[:0]
	n.mux.Unlock()
}

func (n *NativeClock) Active() bool { return true }

func (n *NativeClock) Now() time.Time {
	return time.Now()
}

func (n *NativeClock) Jump(_ time.Duration) {}

func (n *NativeClock) Schedule(d time.Duration, fn func()) {
	ctx, cancel := context.WithCancel(context.Background())
	n.mux.Lock()
	n.cancel = append(n.cancel, cancel)
	n.mux.Unlock()
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
