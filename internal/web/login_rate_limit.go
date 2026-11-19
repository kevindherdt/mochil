package web

import "time"

type loginAttempts struct {
	started time.Time
	count   int
}

func (a *sessionAuth) rateLimited(address string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	a.pruneLocked(now)
	attempt := a.failures[address]
	return attempt.count >= 10 && now.Sub(attempt.started) < 15*time.Minute
}

func (a *sessionAuth) recordFailure(address string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	attempt := a.failures[address]
	if attempt.started.IsZero() || now.Sub(attempt.started) >= 15*time.Minute {
		attempt = loginAttempts{started: now}
	}
	attempt.count++
	a.failures[address] = attempt
}
