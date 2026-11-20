package transcode

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

type cacheStat struct {
	size int64
	used time.Time
}

// evict bounds the persistent segment cache while retaining the conversion
// that just completed and sessions still serving or producing media.
//
// It no longer re-walks every session's directory tree on every call: that
// made cache maintenance cost proportional to the size of the *entire*
// cache on every single finished job. Instead the engine keeps a running
// total and a per-session size cache. The first call seeds that cache with
// one full walk; every later call only re-measures the one directory that
// could have just changed (the session that owns keepKey) and otherwise
// works off the cached numbers for every other session.
func (e *Engine) evict(keepKey string) {
	e.statsMu.Lock()
	if !e.statsReady {
		e.seedCacheStatsLocked()
	} else {
		e.refreshCacheStatLocked(keepKey)
	}
	if e.statsTotal <= e.maxBytes {
		e.statsMu.Unlock()
		return
	}
	type candidate struct {
		key, path string
		size      int64
		used      time.Time
	}
	candidates := make([]candidate, 0, len(e.statsEntries))
	for key, stat := range e.statsEntries {
		candidates = append(candidates, candidate{key: key, path: filepath.Join(e.root, key), size: stat.size, used: stat.used})
	}
	total := e.statsTotal
	e.statsMu.Unlock()

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].used.Before(candidates[j].used) })
	for _, c := range candidates {
		if total <= e.maxBytes {
			break
		}
		if c.key == keepKey {
			continue
		}
		e.mu.Lock()
		current := e.sessions[c.key]
		e.mu.Unlock()
		if current != nil {
			current.mu.Lock()
			protected := current.jobKey != "" || time.Since(current.lastUsed) < e.readyAfter
			current.mu.Unlock()
			if protected {
				continue
			}
		}
		if os.RemoveAll(c.path) == nil {
			total -= c.size
			e.mu.Lock()
			delete(e.sessions, c.key)
			e.mu.Unlock()
			e.statsMu.Lock()
			// Another evict may have removed this session since our snapshot (RemoveAll
			// succeeds on a missing directory); subtract only if the entry is still ours.
			if _, present := e.statsEntries[c.key]; present {
				delete(e.statsEntries, c.key)
				e.statsTotal -= c.size
			}
			e.statsMu.Unlock()
		}
	}
}

// seedCacheStatsLocked performs the one full-tree walk needed to learn the
// current on-disk size of every session. Call with statsMu held.
func (e *Engine) seedCacheStatsLocked() {
	e.statsEntries = map[string]cacheStat{}
	e.statsTotal = 0
	directories, _ := os.ReadDir(e.root)
	for _, directory := range directories {
		if !directory.IsDir() {
			continue
		}
		info, err := directory.Info()
		if err != nil {
			continue
		}
		size := directorySize(filepath.Join(e.root, directory.Name()))
		e.statsEntries[directory.Name()] = cacheStat{size: size, used: e.sessionUsedTime(directory.Name(), info.ModTime())}
		e.statsTotal += size
	}
	e.statsReady = true
}

// refreshCacheStatLocked re-measures only the single session directory that
// could have just changed size, instead of the whole cache tree. Call with
// statsMu held.
func (e *Engine) refreshCacheStatLocked(key string) {
	path := filepath.Join(e.root, key)
	size := directorySize(path)
	used := time.Now()
	if info, err := os.Stat(path); err == nil {
		used = info.ModTime()
	}
	used = e.sessionUsedTime(key, used)
	previous, existed := e.statsEntries[key]
	if existed {
		e.statsTotal += size - previous.size
	} else {
		e.statsTotal += size
	}
	e.statsEntries[key] = cacheStat{size: size, used: used}
}

func (e *Engine) sessionUsedTime(key string, fallback time.Time) time.Time {
	e.mu.Lock()
	current := e.sessions[key]
	e.mu.Unlock()
	if current == nil {
		return fallback
	}
	current.mu.Lock()
	defer current.mu.Unlock()
	if current.lastUsed.After(fallback) {
		return current.lastUsed
	}
	return fallback
}

func directorySize(path string) int64 {
	var size int64
	_ = filepath.WalkDir(path, func(_ string, item os.DirEntry, err error) error {
		if err == nil && !item.IsDir() {
			if info, statErr := item.Info(); statErr == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size
}
