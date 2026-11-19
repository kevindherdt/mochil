// Package filefacts keeps something learned by reading a book's file (an ISBN found in it, a match made for it) for as
// long as the file stays the same. If its size or date changes, what was kept is stale and is ignored.
package filefacts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type entry[T any] struct {
	Value    T         `json:"value"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

type fileData[T any] struct {
	Version int                            `json:"version"`
	Books   map[string]map[string]entry[T] `json:"books"` // library -> book -> fact
}

type Store[T any] struct {
	mu   sync.Mutex
	path string
	data fileData[T]
}

// Open reads the kept facts; an empty path keeps them in memory only.
func Open[T any](path string) (*Store[T], error) {
	store := &Store[T]{path: path, data: fileData[T]{Version: 1, Books: map[string]map[string]entry[T]{}}}
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
		return nil, errors.New("the stored facts in " + filepath.Base(path) + " are unreadable")
	}
	if store.data.Books == nil {
		store.data.Books = map[string]map[string]entry[T]{}
	}
	return store, nil
}

// Get returns what was learned about this version of a book's file.
func (s *Store[T]) Get(library, book string, size int64, modified time.Time) (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, ok := s.data.Books[library][book]
	if !ok || stored.Size != size || !stored.Modified.Equal(modified) {
		var zero T
		return zero, false
	}
	return stored.Value, true
}

func (s *Store[T]) Put(library, book string, size int64, modified time.Time, value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Books[library] == nil {
		s.data.Books[library] = map[string]entry[T]{}
	}
	s.data.Books[library][book] = entry[T]{Value: value, Size: size, Modified: modified}
}

// Remove forgets what was learned about one book, so the next lookup redoes it.
// The rest of the cache is untouched.
func (s *Store[T]) Remove(library, book string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	books, ok := s.data.Books[library]
	if !ok {
		return false
	}
	if _, ok := books[book]; !ok {
		return false
	}
	delete(books, book)
	return true
}

// saveTestHook lets tests observe Save() while it still holds the lock, mid-write -- nil and free in
// production. It exists to prove the lock covers the entire write, not just the marshal.
var saveTestHook func()

func (s *Store[T]) Save() error {
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
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".facts-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	// Flush before the rename: a rename can be durable while the data pages are not,
	// which is how a crash leaves a zero-length file that Open() refuses to start on.
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), s.path)
}
