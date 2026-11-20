package web

import (
	"sync"
	"time"
)

// videoViewers notices who recently fetched an HLS stream for a key, so a stop
// request from one player cannot cancel an encode another account is watching.
// One encoder serves every viewer of the same episode, and the passive idle
// timeout already finishes a job nobody is fetching.
//
// Every account that fetched recently is remembered, not just the last one: a
// player that closes has by definition just been fetching segments, so keeping
// only the most recent viewer made its own stop look unshared.
type videoViewers struct {
	mu   sync.Mutex
	last map[string]map[string]time.Time
}

// videoViewerWindow is how recently another viewer must have fetched for the
// stream to count as shared.
const videoViewerWindow = 30 * time.Second

func (v *videoViewers) observe(key, viewer string) {
	if key == "" || viewer == "" {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.last == nil {
		v.last = make(map[string]map[string]time.Time)
	}
	seen := v.last[key]
	if seen == nil {
		seen = make(map[string]time.Time)
		v.last[key] = seen
	}
	now := time.Now()
	seen[viewer] = now
	for other, at := range seen {
		if now.Sub(at) > videoViewerWindow {
			delete(seen, other)
		}
	}
	if len(seen) == 0 {
		delete(v.last, key)
	}
}

// remembered reports whether a viewer other than this one fetched the key
// recently, in which case stopping the encoder would interrupt them.
func (v *videoViewers) remembered(key, viewer string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now()
	for other, at := range v.last[key] {
		if other != viewer && now.Sub(at) <= videoViewerWindow {
			return true
		}
	}
	return false
}
