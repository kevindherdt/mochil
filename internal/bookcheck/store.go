// Package bookcheck remembers what a quality check found in each book's file, so a library is read once and the
// findings are kept until the file changes.
package bookcheck

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// A finding belongs to one version of a file: if its size or date changes, it is stale and is ignored.
type entry struct {
	Problems []string  `json:"problems,omitempty"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

type fileData struct {
	Version int                         `json:"version"`
	Books   map[string]map[string]entry `json:"books"` // library -> book -> finding
}

type Store struct {
	mu   sync.Mutex
	path string
	data fileData
}

// Open reads the stored findings; an empty path keeps them in memory only.
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
		return nil, errors.New("the stored quality findings are unreadable")
	}
	if store.data.Books == nil {
		store.data.Books = map[string]map[string]entry{}
	}
	return store, nil
}

// Get returns what was found for this version of a book's file, and whether it was checked at all.
func (s *Store) Get(library, book string, size int64, modified time.Time) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.data.Books[library][book]
	if !ok || stored.Size != size || !stored.Modified.Equal(modified) {
		return nil, false
	}
	return append([]string{}, stored.Problems...), true
}

// Put remembers a finding (no problems is a finding too: the book was checked and is fine).
func (s *Store) Put(library, book string, size int64, modified time.Time, problems []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Books[library] == nil {
		s.data.Books[library] = map[string]entry{}
	}
	s.data.Books[library][book] = entry{Problems: problems, Size: size, Modified: modified}
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
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".book-quality-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	// Flush before the rename so a crash cannot leave an empty file the loader rejects.
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), s.path)
}
