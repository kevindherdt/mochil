package web

import (
	"sync"
	"time"
)

// jellyfinLimiter stops password guessing through the Jellyfin sign-in route, which is outside the web login's
// own lock-out: ten wrong passwords from one address block that address for fifteen minutes.
type jellyfinLimiter struct {
	mu       sync.Mutex
	failures map[string]loginAttempts
	now      func() time.Time
}

const (
	jellyfinFailureLimit  = 10
	jellyfinFailureWindow = 15 * time.Minute
	jellyfinFailureTable  = 4096
)

func (l *jellyfinLimiter) clock() time.Time {
	if l.now != nil {
		return l.now()
	}
	return time.Now()
}

// blocked reports whether the address has used up its wrong guesses.
func (l *jellyfinLimiter) blocked(address string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt := l.failures[address]
	return attempt.count >= jellyfinFailureLimit && l.clock().Sub(attempt.started) < jellyfinFailureWindow
}

func (l *jellyfinLimiter) fail(address string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	if l.failures == nil {
		l.failures = map[string]loginAttempts{}
	}
	if len(l.failures) >= jellyfinFailureTable {
		for key, attempt := range l.failures {
			if now.Sub(attempt.started) >= jellyfinFailureWindow {
				delete(l.failures, key)
			}
		}
		if len(l.failures) >= jellyfinFailureTable {
			return // a flood of addresses: keep what is already counted rather than grow without bound
		}
	}
	attempt := l.failures[address]
	if attempt.started.IsZero() || now.Sub(attempt.started) >= jellyfinFailureWindow {
		attempt = loginAttempts{started: now}
	}
	attempt.count++
	l.failures[address] = attempt
}

func (l *jellyfinLimiter) clear(address string) {
	l.mu.Lock()
	delete(l.failures, address)
	l.mu.Unlock()
}
