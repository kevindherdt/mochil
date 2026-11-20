package bookstats

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// A stored count belongs to one version of a file: if its size or date changes, the count is stale and is ignored.
type entry struct {
	Stats
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

type fileData struct {
	Version int                         `json:"version"`
	Books   map[string]map[string]entry `json:"books"` // library -> book -> count
}

// Store remembers counts between runs. Counting a library reads every book, so it is done once and kept.
type Store struct {
	mu   sync.Mutex
	path string
	data fileData
}

// Open reads the stored counts; an empty path keeps them in memory only.
func Open(path string) (*Store, error) {
	store := &Store{path: path, data: fileData{Version: 1, Books: map[string]map[string]entry{}}}
	if path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &store.data); err != nil || store.data.Version != 1 {
		return nil, errors.New("the stored book counts are unreadable")
	}
	if store.data.Books == nil {
		store.data.Books = map[string]map[string]entry{}
	}
	return store, nil
}

// Get returns the count for a book, if one was made for this version of its file.
func (s *Store) Get(library, book string, size int64, modified time.Time) (Stats, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.data.Books[library][book]
	if !ok || stored.Size != size || !stored.Modified.Equal(modified) {
		return Stats{}, false
	}
	return stored.Stats, true
}

// Put remembers a count. Save writes them out; counting many books saves once at the end.
func (s *Store) Put(library, book string, size int64, modified time.Time, stats Stats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Books[library] == nil {
		s.data.Books[library] = map[string]entry{}
	}
	s.data.Books[library][book] = entry{Stats: stats, Size: size, Modified: modified}
}

// saveTestHook lets tests observe Save() while it still holds the lock, mid-write -- nil and free in
// production. It exists to prove the lock covers the entire write, not just the marshal.
var saveTestHook func()

func (s *Store) Save() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	if saveTestHook != nil {
		saveTestHook()
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".book-stats-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	// Flush before the rename: a rename can be durable while the data pages are
	// not, which is how a crash leaves a zero-length file that Open() refuses to
	// start on.
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), s.path)
}
