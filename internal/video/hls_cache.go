package video

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// highestContiguousSegment identifies the generated range an active encoder
// can satisfy without restarting at a distant seek position.
func highestContiguousSegment(directory string, start int) int {
	for segment := start; ; segment++ {
		info, err := os.Stat(filepath.Join(directory, fmt.Sprintf("seg%05d.ts", segment)))
		if err != nil || info.Size() == 0 {
			return segment - 1
		}
	}
}

func completePlaylist(name string) bool {
	data, err := os.ReadFile(name)
	return err == nil && strings.Contains(string(data), "#EXT-X-ENDLIST")
}

type hlsCacheStat struct {
	size int64
	used time.Time
}

// evict bounds the persistent segment cache while retaining the conversion
// that just completed and sessions still serving or producing media.
//
// It no longer re-walks every session's directory tree on every call: that
// made cache maintenance cost proportional to the size of the *entire*
// cache on every single finished job. Instead the store keeps a running
// total and a per-session size cache. The first call seeds that cache with
// one full walk; every later call only re-measures the one directory that
// could have just changed (the session that owns keepKey) and otherwise
// works off the cached numbers for every other session.
func (s *HLSStore) evict(keepKey string) {
	s.statsMu.Lock()
	if !s.statsReady {
		s.seedCacheStatsLocked()
	} else {
		s.refreshCacheStatLocked(keepKey)
	}
	if s.statsTotal <= s.maxBytes {
		s.statsMu.Unlock()
		return
	}
	type candidate struct {
		key, path string
		size      int64
		used      time.Time
	}
	candidates := make([]candidate, 0, len(s.statsEntries))
	for key, stat := range s.statsEntries {
		candidates = append(candidates, candidate{key: key, path: filepath.Join(s.root, key), size: stat.size, used: stat.used})
	}
	total := s.statsTotal
	s.statsMu.Unlock()

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].used.Before(candidates[j].used) })
	for _, candidate := range candidates {
		if total <= s.maxBytes {
			break
		}
		if candidate.key == keepKey {
			continue
		}
		s.mu.Lock()
		session := s.sessions[candidate.key]
		s.mu.Unlock()
		if session != nil {
			session.mu.Lock()
			// Queued work counts as active: jobKey is set before the job is submitted,
			// while process only appears once it starts running. Evicting a session
			// that was still waiting for an encoder deleted the directory its ffmpeg
			// was about to write into.
			active := session.process != nil || session.jobKey != "" || time.Since(session.lastUsed) < s.readyAfter
			session.mu.Unlock()
			if active {
				continue
			}
		}
		if os.RemoveAll(candidate.path) == nil {
			total -= candidate.size
			s.mu.Lock()
			delete(s.sessions, candidate.key)
			s.mu.Unlock()
			s.statsMu.Lock()
			// Another evict may have removed this session since our snapshot (RemoveAll
			// succeeds on a missing directory); subtract only if the entry is still ours.
			if _, present := s.statsEntries[candidate.key]; present {
				delete(s.statsEntries, candidate.key)
				s.statsTotal -= candidate.size
			}
			s.statsMu.Unlock()
		}
	}
}

// seedCacheStatsLocked performs the one full-tree walk needed to learn the
// current on-disk size of every session. Call with statsMu held.
func (s *HLSStore) seedCacheStatsLocked() {
	s.statsEntries = map[string]hlsCacheStat{}
	s.statsTotal = 0
	directories, _ := os.ReadDir(s.root)
	for _, directory := range directories {
		if !directory.IsDir() {
			continue
		}
		info, err := directory.Info()
		if err != nil {
			continue
		}
		size := hlsDirectorySize(filepath.Join(s.root, directory.Name()))
		s.statsEntries[directory.Name()] = hlsCacheStat{size: size, used: s.sessionUsedTime(directory.Name(), info.ModTime())}
		s.statsTotal += size
	}
	s.statsReady = true
}

// refreshCacheStatLocked re-measures only the single session directory that
// could have just changed size, instead of the whole cache tree. Call with
// statsMu held.
func (s *HLSStore) refreshCacheStatLocked(key string) {
	path := filepath.Join(s.root, key)
	size := hlsDirectorySize(path)
	used := time.Now()
	if info, err := os.Stat(path); err == nil {
		used = info.ModTime()
	}
	used = s.sessionUsedTime(key, used)
	previous, existed := s.statsEntries[key]
	if existed {
		s.statsTotal += size - previous.size
	} else {
		s.statsTotal += size
	}
	s.statsEntries[key] = hlsCacheStat{size: size, used: used}
}

func (s *HLSStore) sessionUsedTime(key string, fallback time.Time) time.Time {
	s.mu.Lock()
	session := s.sessions[key]
	s.mu.Unlock()
	if session == nil {
		return fallback
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.lastUsed.After(fallback) {
		return session.lastUsed
	}
	return fallback
}

func hlsDirectorySize(path string) int64 {
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
