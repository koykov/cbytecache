package cbytecache

import (
	"testing"
	"time"
)

func TestConfigCopy(t *testing.T) {
	conf := DefaultConfig(time.Minute, nil)
	cpy := conf.Copy()
	conf.ExpireInterval = 30 * time.Second
	if cpy.ExpireInterval != time.Minute {
		t.Error("config copy failed")
	}
}
