// Package safely keeps a failure in one piece of background work from taking the whole server down.
package safely

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"sync"
	"time"
)

// Failure is what Settings shows about a piece of background work that stopped with a panic.
type Failure struct {
	What     string `json:"what"`
	At       int64  `json:"at"` // unix milliseconds of the latest one
	Message  string `json:"message"`
	Count    int    `json:"count"`
	Restarts int    `json:"restarts"`
}

var (
	mu       sync.Mutex
	failures = map[string]*Failure{}
)

func record(what string, recovered any, restarted bool) {
	message := fmt.Sprint(recovered)
	if len(message) > 200 {
		message = message[:200] + "…"
	}
	mu.Lock()
	defer mu.Unlock()
	entry := failures[what]
	if entry == nil {
		entry = &Failure{What: what}
		failures[what] = entry
	}
	entry.At, entry.Message = time.Now().UnixMilli(), message
	entry.Count++
	if restarted {
		entry.Restarts++
	}
}

// Failures lists the work that has panicked since the server started, most recent first.
func Failures() []Failure {
	mu.Lock()
	defer mu.Unlock()
	list := make([]Failure, 0, len(failures))
	for _, entry := range failures {
		list = append(list, *entry)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].At > list[j].At })
	return list
}

// Recover is for "defer safely.Recover(...)" at the top of a goroutine: a panic is logged and ends only that goroutine.
func Recover(what string) {
	if recovered := recover(); recovered != nil {
		slog.Error("background work panicked", "work", what, "panic", recovered, "stack", string(debug.Stack()))
		record(what, recovered, false)
	}
}

// Run does work that no request is waiting on, containing a panic the same way.
func Run(what string, work func()) {
	defer Recover(what)
	work()
}

// Delays between a worker's restarts; a var so tests need not wait.
var (
	firstDelay   = 5 * time.Second
	longestDelay = 5 * time.Minute
	healthyFor   = time.Minute
)

// Keep runs a long-lived worker and starts it again after a panic, waiting longer each time it fails quickly so a
// worker that always fails does not spin. It returns when the work returns normally or ctx ends.
func Keep(ctx context.Context, what string, work func()) {
	delay := firstDelay
	for {
		started := time.Now()
		if panicked := attempt(what, work); !panicked {
			return
		}
		if time.Since(started) > healthyFor {
			delay = firstDelay
		}
		slog.Warn("background work restarting", "work", what, "in", delay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		mu.Lock()
		failures[what].Restarts++
		mu.Unlock()
		delay = nextDelay(delay)
	}
}

// nextDelay doubles the wait between restarts, up to the longest.
func nextDelay(delay time.Duration) time.Duration { return min(delay*2, longestDelay) }

func attempt(what string, work func()) (panicked bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("background work panicked", "work", what, "panic", recovered, "stack", string(debug.Stack()))
			record(what, recovered, false)
			panicked = true
		}
	}()
	work()
	return false
}
