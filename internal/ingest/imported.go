package ingest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ImportedStore remembers which drop-folder files have been copied, so every device sees the same "copied" marks.
// It is one small file next to the journal, shared by the managed folder and every destination library.
type ImportedStore struct {
	path string
	mu   sync.Mutex
	set  map[string]struct{}
}

func newImportedStore(journalPath string) *ImportedStore {
	store := &ImportedStore{path: strings.TrimSuffix(journalPath, ".json") + "-imported.json", set: map[string]struct{}{}}
	if data, err := os.ReadFile(store.path); err == nil {
		var paths []string
		if json.Unmarshal(data, &paths) == nil {
			for _, p := range paths {
				store.set[p] = struct{}{}
			}
		}
	}
	return store
}

// Mark records copied source paths; it is idempotent and best effort (a lost mark only means a book is offered again).
func (s *ImportedStore) Mark(paths ...string) {
	if s == nil || len(paths) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for _, p := range paths {
		if _, ok := s.set[p]; !ok {
			s.set[p] = struct{}{}
			changed = true
		}
	}
	if !changed {
		return
	}
	all := make([]string, 0, len(s.set))
	for p := range s.set {
		all = append(all, p)
	}
	data, err := json.Marshal(all)
	if err != nil || os.MkdirAll(filepath.Dir(s.path), 0o700) != nil {
		return
	}
	temporary := s.path + ".tmp"
	if os.WriteFile(temporary, data, 0o600) == nil {
		_ = os.Rename(temporary, s.path)
	}
}

// Has reports whether a source path was copied.
func (s *ImportedStore) Has(path string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.set[path]
	return ok
}

func (e *Executor) Imported() *ImportedStore { return e.imported }

// seedImported marks everything this journal already committed, so imports made before the store existed still show.
func (e *Executor) seedImported() {
	journal, err := e.journal.Load()
	if err != nil {
		return
	}
	var paths []string
	for _, transaction := range journal.Transactions {
		if transaction.State == TransactionCommitted {
			for _, entry := range transaction.Entries {
				paths = append(paths, entry.SourcePath)
			}
		}
	}
	e.imported.Mark(paths...)
}
