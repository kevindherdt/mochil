package web

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/enrichment"
	"mochil/internal/statefile"
)

// Why a book is not being sent to Hardcover.
const (
	ignoredRemoved = "removed on Hardcover" // it was on the shelf before and is gone, so the reader took it off
	ignoredByYou   = "stopped by you"
)

type ignoredBook struct {
	Library string   `json:"library"`
	Book    string   `json:"book"`
	Title   string   `json:"title"`
	Authors []string `json:"authors,omitempty"`
	Reason  string   `json:"reason"`
	At      int64    `json:"at"`
}

// hardcoverIgnored is the list of books the shelf sync leaves alone: ones taken off the Hardcover shelf, and ones
// the reader asked to stop syncing. Reading them changes nothing on Hardcover until they are synced again.
type hardcoverIgnored struct {
	mu    sync.Mutex
	path  string
	items []ignoredBook
}

func (s *hardcoverIgnored) open(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
	if path == "" {
		return nil
	}
	var stored struct {
		Items []ignoredBook `json:"items"`
	}
	if err := statefile.DecodeJSON(path, 1<<20, &stored); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.items = stored.Items
	return nil
}

func (s *hardcoverIgnored) has(libraryID, bookID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		if item.Library == libraryID && item.Book == bookID {
			return true
		}
	}
	return false
}

func (s *hardcoverIgnored) add(item ignoredBook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.Library == item.Library && existing.Book == item.Book {
			return
		}
	}
	s.items = append([]ignoredBook{item}, s.items...)
	s.saveLocked()
}

func (s *hardcoverIgnored) remove(libraryID, bookID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.items {
		if existing.Library == libraryID && existing.Book == bookID {
			s.items = append(s.items[:i:i], s.items[i+1:]...)
			s.saveLocked()
			return
		}
	}
}

func (s *hardcoverIgnored) list() []ignoredBook {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ignoredBook{}, s.items...)
}

func (s *hardcoverIgnored) saveLocked() {
	if s.path != "" {
		_ = writeJSONFile(s.path, map[string]any{"items": s.items}, ".hardcover-ignored-*")
	}
}

// onShelfBefore is true when the shelf sync has already put this book on the Hardcover shelf, so finding it absent
// later means the reader removed it there.
func onShelfBefore(match hardcoverMatch, had bool) bool {
	if !had {
		return false
	}
	switch enrichment.ShelfResult(match.Result) {
	case enrichment.ShelfAdded, enrichment.ShelfChanged, enrichment.ShelfUnchanged:
		return true
	}
	return false
}

// setHardcoverIgnored stops or resumes sending a book's reading to Hardcover.
func (m *multiServer) setHardcoverIgnored(lib Library, book catalog.Book, ignore bool) {
	if ignore {
		m.hardcoverIgnored.add(ignoredBook{Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...), Reason: ignoredByYou, At: time.Now().Unix()})
		return
	}
	m.hardcoverIgnored.remove(lib.ID, book.ID)
	// Syncing again may put the book back on the shelf, which is what resuming means.
	m.hardcoverMatches.forgetShelf(lib.ID, book.ID)
}

// apiHardcoverIgnore serves /api/metadata/hardcover/ignored (the list) and /ignore (stop or resume syncing a book).
func (m *multiServer) apiHardcoverIgnore(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/ignored") {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": m.hardcoverIgnored.list()})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Library string `json:"library"`
		Book    string `json:"book"`
		Ignore  bool   `json:"ignore"`
	}
	if err := decodeJSONRequest(w, r, 4<<10, &input, true); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	lib, book, ok := m.findBook(input.Library + "/" + input.Book)
	if !ok || !publicationKind(lib.Kind) {
		http.NotFound(w, r)
		return
	}
	m.setHardcoverIgnored(lib, book, input.Ignore)
	writeJSON(w, http.StatusOK, map[string]any{"ignored": input.Ignore})
}
