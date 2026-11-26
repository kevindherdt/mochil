package web

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"mochil/internal/statefile"
)

// taskDismissalVersion is the shape of the dismissal file. A file from another version is refused rather
// than half-read, because a misread here would show results the person already discarded.
const taskDismissalVersion = 1

// taskDismissalFile is what is stored on disk: book ids per library and per tool.
//
// It is nested rather than a flat list of (library, tool, book) rows because every read is a count for one
// tool in one library, and the whole file stays small: at most three lists per library, holding only books
// whose results somebody discarded.
type taskDismissalFile struct {
	Version int                            `json:"version"`
	Books   map[string]map[string][]string `json:"books"` // library -> tool -> book ids
}

// taskDismissals is the list of results the person discarded, kept per library and per tool.
//
// It exists because one shared list was the bug: discarding the found ISBNs also hid those same books from
// the Hardcover matches and from the title cleanup, so a book discarded once looked as if no other tool
// had found anything for it. A tool now only ever sees its own discards.
type taskDismissals struct {
	mu    sync.Mutex
	path  string
	books map[string]map[string][]string
}

func openTaskDismissals(path string) (*taskDismissals, error) {
	store := &taskDismissals{path: path, books: map[string]map[string][]string{}}
	if path == "" {
		return store, nil
	}
	var stored taskDismissalFile
	if err := statefile.DecodeJSON(path, 8<<20, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store, nil
		}
		return nil, err
	}
	if stored.Version != taskDismissalVersion {
		return nil, errors.New("the stored task dismissals in " + filepath.Base(path) + " were written by another version")
	}
	if stored.Books != nil {
		store.books = stored.Books
	}
	return store, nil
}

// count is how many results this tool has had discarded in this library.
func (s *taskDismissals) count(library, tool string) int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.books[library][tool])
}

// add remembers book ids as discarded for one tool in one library. Discarding the same book twice changes
// nothing, so a repeated request neither grows the file nor moves a count.
func (s *taskDismissals) add(library, tool string, books []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(books) == 0 {
		return nil
	}
	previous, had := s.books[library][tool]
	kept := previous
	known := make(map[string]bool, len(books))
	for _, id := range kept {
		known[id] = true
	}
	added := 0
	for _, id := range books {
		if id == "" || known[id] {
			continue
		}
		known[id] = true
		kept = append(kept, id)
		added++
	}
	if added == 0 {
		return nil
	}
	if s.books[library] == nil {
		s.books[library] = map[string][]string{}
	}
	s.books[library][tool] = kept
	if err := s.saveLocked(); err != nil {
		// The count is what the panel shows, so it may not claim more than the file holds: on the next
		// start the file would disagree and the discarded results would come back.
		if had {
			s.books[library][tool] = previous
		} else {
			delete(s.books[library], tool)
		}
		return err
	}
	return nil
}

// clear brings back every result this tool discarded in one library.
func (s *taskDismissals) clear(library, tool string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, had := s.books[library][tool]
	if !had {
		return nil
	}
	delete(s.books[library], tool)
	if err := s.saveLocked(); err != nil {
		if s.books[library] == nil {
			s.books[library] = map[string][]string{}
		}
		s.books[library][tool] = previous
		return err
	}
	return nil
}

func (s *taskDismissals) saveLocked() error {
	if s.path == "" {
		return nil
	}
	// writeJSONFile is the package's atomic write: temp file, chmod, sync, close, rename. Writing in place
	// a file that is read at startup is how a crash leaves it empty.
	return writeJSONFile(s.path, taskDismissalFile{Version: taskDismissalVersion, Books: s.books}, ".task-dismissals-*")
}
