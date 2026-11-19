package accounts

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPasswordChecksNeverRunBeyondTheirBound(t *testing.T) {
	previous := compareHash
	var running, peak atomic.Int32
	compareHash = func(_, _ []byte) error {
		now := running.Add(1)
		for {
			seen := peak.Load()
			if now <= seen || peak.CompareAndSwap(seen, now) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		running.Add(-1)
		return nil
	}
	defer func() { compareHash = previous }()
	var wait sync.WaitGroup
	for range 64 {
		wait.Add(1)
		go func() { defer wait.Done(); _ = comparePassword(nil, nil) }()
	}
	wait.Wait()
	if int(peak.Load()) > cap(passwordChecks) {
		t.Fatalf("%d comparisons ran at once, bound is %d", peak.Load(), cap(passwordChecks))
	}
}
