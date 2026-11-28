package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mochil/internal/statefile"
)

const cacheVersion = 3
const probeVersion = 8
const maximumProbeMigrationsPerScan = 80

type Stats struct{ Items, Reused, Errors int }
type cacheFile struct {
	Version int       `json:"version"`
	Items   []Item    `json:"items"`
	Scanned time.Time `json:"scanned"`
}
type Store struct {
	libraryID, root, cache string
	mu                     sync.RWMutex
	scanMu                 sync.Mutex
	items                  []Item
	byID                   map[string]Item
	prober                 Prober
	version                uint64
}

func NewStore(libraryID, root, cache string) *Store {
	store := &Store{libraryID: libraryID, root: root, cache: cache}
	if cached := store.load(); len(cached) > 0 {
		store.publish(cached)
	}
	return store
}
func (s *Store) SetProber(p Prober) { s.prober = p }
func (s *Store) Items() []Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneItems(s.items)
}
func (s *Store) Snapshot() ([]Item, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneItems(s.items), s.version
}
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
func (s *Store) Version() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// IDs lists the catalogue's item ids without copying item data; see the video
// store's equivalent for why a single-item lookup needs only the keys.
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
	v, ok := s.byID[id]
	return cloneItem(v), ok
}

func (s *Store) Scan() (Stats, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	root, err := filepath.Abs(s.root)
	if err != nil {
		return Stats{}, err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		if cached := s.load(); len(cached) > 0 {
			s.publish(cached)
			return Stats{Items: len(cached), Reused: len(cached), Errors: 1}, nil
		}
		return Stats{}, fmt.Errorf("audio library is unavailable: %s", root)
	}
	previous := map[string]Item{}
	for _, item := range s.Items() {
		previous[item.Path] = item
	}
	if len(previous) == 0 {
		for _, item := range s.load() {
			previous[item.Path] = item
		}
	}
	items := []Item{}
	stats := Stats{}
	type probeJob struct {
		index int
		path  string
	}
	jobs := []probeJob{}
	probeMigrations := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		// See internal/video/store.go: a failed walk aborts the scan rather than publishing a partial catalogue.
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
		fileInfo, e := audioScanInfo(entry)
		if e != nil {
			return e
		}
		if fileInfo == nil {
			return nil
		}
		relative, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		item, ok := Identify(s.libraryID, relative, fileInfo.Size(), fileInfo.ModTime())
		if !ok {
			return nil
		}
		reused, migrate := false, false
		if old, exists := previous[item.Path]; exists && old.Revision == item.Revision {
			item = old
			inferFolderMetadata(&item)
			if old.ProbeVersion == probeVersion {
				stats.Reused++
				reused = true
			} else if s.prober == nil || probeMigrations >= maximumProbeMigrationsPerScan {
				stats.Reused++
				reused = true
			} else {
				probeMigrations++
				migrate = true
			}
		}
		items = append(items, item)
		if (migrate || !reused) && s.prober != nil {
			jobs = append(jobs, probeJob{len(items) - 1, path})
		}
		return nil
	})
	if err != nil {
		return Stats{}, err
	}
	if len(jobs) > 0 {
		queue := make(chan probeJob)
		var wait sync.WaitGroup
		var failures atomic.Int64
		workers := 4
		if len(jobs) < workers {
			workers = len(jobs)
		}
		for range workers {
			wait.Add(1)
			go func() {
				defer wait.Done()
				for job := range queue {
					metadata, probeErr := probeContained(s.prober, job.path)
					if probeErr != nil {
						// Leave the version alone: stamping it here claimed current probe data for an item that has none, so
						// every later scan reused it and the file was never probed again until its size or mtime changed.
						failures.Add(1)
						continue
					}
					applyMetadata(&items[job.index], metadata)
				}
			}()
		}
		for _, job := range jobs {
			queue <- job
		}
		close(queue)
		wait.Wait()
		stats.Errors = int(failures.Load())
	}
	for index := range items {
		inferFolderMetadata(&items[index])
	}
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		// EqualFold, not ToLower: case folding is Unicode-aware and comparing the
		// folded forms keeps sort and grouping agreeing on the same strings.
		if !strings.EqualFold(left.AlbumArtist, right.AlbumArtist) {
			return strings.ToLower(left.AlbumArtist) < strings.ToLower(right.AlbumArtist)
		}
		if !strings.EqualFold(left.Album, right.Album) {
			return strings.ToLower(left.Album) < strings.ToLower(right.Album)
		}
		if left.Disc != right.Disc {
			return left.Disc < right.Disc
		}
		if left.Track != right.Track {
			return left.Track < right.Track
		}
		return strings.ToLower(left.Title) < strings.ToLower(right.Title)
	})
	// Publish before saving: losing the whole catalogue because the cache file could not be written turns a full disk into
	// an empty library.
	s.publish(items)
	stats.Items = len(items)
	if err = s.save(items); err != nil {
		return stats, err
	}
	return stats, nil
}
func applyMetadata(item *Item, m Metadata) {
	if m.Title != "" {
		item.Title = m.Title
	}
	item.Artist = m.Artist
	item.AlbumArtist = m.AlbumArtist
	item.Album = m.Album
	item.Series = m.Series
	item.SeriesIndex = m.SeriesIndex
	item.Genre = m.Genre
	item.Narrator = m.Narrator
	item.Track = m.Track
	item.Disc = m.Disc
	item.Year = m.Year
	item.Bitrate = m.Bitrate
	item.Codec = m.Codec
	item.SampleRate = m.SampleRate
	item.BitDepth = m.BitDepth
	item.HasArtwork = m.HasArtwork
	item.Duration = m.Duration
	item.ReplayTrack = m.ReplayTrack
	item.ReplayAlbum = m.ReplayAlbum
	item.PeakTrack = m.PeakTrack
	item.PeakAlbum = m.PeakAlbum
	item.Chapters = append([]Chapter(nil), m.Chapters...)
	item.ProbeVersion = probeVersion
	inferFolderMetadata(item)
}

func (s *Store) publish(items []Item) {
	byID := make(map[string]Item, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	s.mu.Lock()
	s.items = append([]Item(nil), items...)
	s.byID = byID
	s.version++
	s.mu.Unlock()
}
func (s *Store) load() []Item {
	var cached cacheFile
	if statefile.DecodeJSON(s.cache, statefile.MaximumCatalogueBytes, &cached) != nil || (cached.Version < 1 || cached.Version > cacheVersion) || len(cached.Items) > 1_000_000 {
		return nil
	}
	return cached.Items
}
func (s *Store) save(items []Item) error {
	if s.cache == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.cache), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.cache), ".audio-cache-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(temporary, statefile.MaximumCatalogueBytes, cacheFile{cacheVersion, items, time.Now().UTC()}, false); err == nil {
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

func (s *Store) Open(id string) (*os.File, Item, error) {
	item, ok := s.Item(id)
	if !ok {
		return nil, Item{}, os.ErrNotExist
	}
	root, _ := filepath.Abs(s.root)
	candidate := filepath.Join(root, filepath.FromSlash(item.Path))
	resolvedRoot, e := filepath.EvalSymlinks(root)
	if e != nil {
		return nil, Item{}, e
	}
	resolved, e := filepath.EvalSymlinks(candidate)
	if e != nil {
		return nil, Item{}, e
	}
	relative, e := filepath.Rel(resolvedRoot, resolved)
	if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, Item{}, os.ErrPermission
	}
	file, e := os.Open(resolved)
	return file, item, e
}

// probeContained keeps a parser panic on one malformed file from ending the scan.
func probeContained(prober Prober, path string) (metadata Metadata, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("probe panicked: %v", recovered)
		}
	}()
	return prober.Probe(path)
}
