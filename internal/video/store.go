package video

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"mochil/internal/statefile"
)

const cacheVersion = 2

// probeVersion is bumped whenever the probe's answers change shape or meaning, which re-probes what is already cached.
const probeVersion = 1

type Stats struct {
	Items  int
	Reused int
	Errors int
}

type cacheFile struct {
	Version int       `json:"version"`
	Items   []Item    `json:"items"`
	Scanned time.Time `json:"scanned"`
}

type Store struct {
	libraryID string
	root      string
	cache     string
	mu        sync.RWMutex
	scanMu    sync.Mutex
	items     []Item
	byID      map[string]Item
	version   uint64
	prober    Prober
}

func NewStore(libraryID, root, cache string) *Store {
	return &Store{libraryID: libraryID, root: root, cache: cache}
}

func (s *Store) SetProber(prober Prober) { s.prober = prober }

// probeStaleForTest ages the published items' probe stamp, which is what a bumped probeVersion looks like to a scan.
func (s *Store) probeStaleForTest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.items {
		s.items[index].ProbeVersion = probeVersion - 1
	}
}

func (s *Store) Items() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneItems(s.items)
}

// Count returns the current snapshot size without copying the catalogue.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// IDs lists the catalogue's item ids without copying any item data. A single-item
// lookup (Jellyfin maps its own ids through a hash of ours, so it cannot invert
// them) only needs the keys: cloning every item to compare one id turned each
// playback and progress request into a full catalogue copy.
func (s *Store) IDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.byID))
	for id := range s.byID {
		ids = append(ids, id)
	}
	return ids
}

func (s *Store) Item(id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.byID[id]
	return cloneItem(item), ok
}

func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

func (s *Store) Scan() (Stats, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	root, err := filepath.Abs(s.root)
	if err != nil {
		return Stats{}, err
	}
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		if s.hasSnapshot() {
			return Stats{Items: len(s.Items()), Reused: len(s.Items()), Errors: 1}, nil
		}
		if cached := s.loadCache(); len(cached) > 0 {
			s.publish(cached)
			return Stats{Items: len(cached), Reused: len(cached), Errors: 1}, nil
		}
		if statErr != nil {
			return Stats{}, fmt.Errorf("video library: %w", statErr)
		}
		return Stats{}, fmt.Errorf("video library is not a directory: %s", root)
	}
	previous := make(map[string]Item)
	previousItems := s.Items()
	if len(previousItems) == 0 {
		previousItems = s.loadCache()
	}
	for _, item := range previousItems {
		previous[item.Path] = item
	}
	items := make([]Item, 0)
	stats := Stats{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		// A walk that fails must abort the scan, not publish a partial catalogue: internal/video's
		// TestStoreDoesNotPublishPartialSnapshotAfterWalkFailure holds that guarantee, and a library half-listed after a
		// permission change is worse than one that keeps its last-known-good snapshot and reports the failure.
		if walkErr != nil {
			return walkErr
		}
		if path != root && entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		item, supported := Identify(s.libraryID, relative, info.Size(), info.ModTime())
		if !supported {
			return nil
		}
		reused := false
		if old, ok := previous[item.Path]; ok && old.Revision == item.Revision && (s.prober == nil || (len(old.Video) > 0 && old.ProbeVersion == probeVersion)) {
			item = old
			stats.Reused++
			reused = true
		}
		if s.prober != nil && !reused {
			probe, probeErr := s.prober.Probe(path)
			if probeErr != nil {
				// No stamp on failure: a stamped version would claim current probe data for a file that has none, and every
				// later scan would reuse it.
				stats.Errors++
			} else {
				item.Duration, item.Video, item.Audio, item.Subtitles = probe.Duration, probe.Video, probe.Audio, probe.Subtitles
				item.ProbeVersion = probeVersion
			}
		}
		InferEpisode(&item)
		items = append(items, item)
		return nil
	})
	if err != nil {
		return Stats{}, err
	}
	sortItems(items)
	// Publish first: the scan did discover these items, and losing the catalogue because the cache file could not be written
	// turns a full disk into an empty library.
	s.publish(items)
	stats.Items = len(items)
	if err := s.writeCache(items); err != nil {
		return stats, err
	}
	return stats, nil
}

func (s *Store) hasSnapshot() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID != nil
}

func (s *Store) publish(items []Item) {
	byID := make(map[string]Item, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	s.mu.Lock()
	s.items, s.byID, s.version = append([]Item(nil), items...), byID, s.version+1
	s.mu.Unlock()
}

func (s *Store) loadCache() []Item {
	var cached cacheFile
	if statefile.DecodeJSON(s.cache, statefile.MaximumCatalogueBytes, &cached) != nil || cached.Version != cacheVersion || len(cached.Items) > 1_000_000 {
		return nil
	}
	return cached.Items
}

func (s *Store) writeCache(items []Item) error {
	if s.cache == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.cache), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.cache), ".video-cache-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(temporary, statefile.MaximumCatalogueBytes, cacheFile{Version: cacheVersion, Items: items, Scanned: time.Now().UTC()}, false); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.cache)
}

func naturalLess(left, right string) bool {
	leftRunes, rightRunes := []rune(left), []rune(right)
	for li, ri := 0, 0; li < len(leftRunes) && ri < len(rightRunes); {
		if leftRunes[li] >= '0' && leftRunes[li] <= '9' && rightRunes[ri] >= '0' && rightRunes[ri] <= '9' {
			lj, rj := li, ri
			for lj < len(leftRunes) && leftRunes[lj] >= '0' && leftRunes[lj] <= '9' {
				lj++
			}
			for rj < len(rightRunes) && rightRunes[rj] >= '0' && rightRunes[rj] <= '9' {
				rj++
			}
			ln, rn := string(leftRunes[li:lj]), string(rightRunes[ri:rj])
			ln, rn = trimZeros(ln), trimZeros(rn)
			if len(ln) != len(rn) {
				return len(ln) < len(rn)
			}
			if ln != rn {
				return ln < rn
			}
			li, ri = lj, rj
			continue
		}
		l, r := leftRunes[li], rightRunes[ri]
		if l >= 'A' && l <= 'Z' {
			l += 'a' - 'A'
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if l != r {
			return l < r
		}
		li, ri = li+1, ri+1
	}
	return len(leftRunes) < len(rightRunes)
}

func trimZeros(value string) string {
	for len(value) > 1 && value[0] == '0' {
		value = value[1:]
	}
	return value
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Show != items[j].Show {
			return naturalLess(items[i].Show, items[j].Show)
		}
		if items[i].Show != "" && items[i].Season != items[j].Season {
			return items[i].Season < items[j].Season
		}
		if items[i].Show != "" && items[i].Episode != items[j].Episode {
			return items[i].Episode < items[j].Episode
		}
		if items[i].Title != items[j].Title {
			return naturalLess(items[i].Title, items[j].Title)
		}
		return items[i].Path < items[j].Path
	})
}
