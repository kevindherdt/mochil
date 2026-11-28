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

// addAll records every item that is not recorded already and writes the file once. The caller with many books used to go
// through add() per book, which meant one encode, fsync and rename each -- and every one of those errors was discarded.
func (s *hardcoverIgnored) addAll(items []ignoredBook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := 0
	for _, item := range items {
		known := false
		for _, existing := range s.items {
			if existing.Library == item.Library && existing.Book == item.Book {
				known = true
				break
			}
		}
		if !known {
			s.items = append([]ignoredBook{item}, s.items...)
			added++
		}
	}
	if added == 0 {
		return nil
	}
	return s.saveLocked()
}

// removeAll drops the listed books and writes the file once.
func (s *hardcoverIgnored) removeAll(items []ignoredBook) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	drop := make(map[string]bool, len(items))
	for _, item := range items {
		drop[item.Library+"\x00"+item.Book] = true
	}
	kept := s.items[:0]
	for _, existing := range s.items {
		if !drop[existing.Library+"\x00"+existing.Book] {
			kept = append(kept, existing)
		}
	}
	if len(kept) == len(s.items) {
		return nil
	}
	s.items = kept
	return s.saveLocked()
}

func (s *hardcoverIgnored) add(item ignoredBook) error { return s.addAll([]ignoredBook{item}) }

func (s *hardcoverIgnored) remove(libraryID, bookID string) error {
	return s.removeAll([]ignoredBook{{Library: libraryID, Book: bookID}})
}

func (s *hardcoverIgnored) list() []ignoredBook {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ignoredBook{}, s.items...)
}

func (s *hardcoverIgnored) saveLocked() error {
	if s.path == "" {
		return nil
	}
	// Returning the error is the point: a UI that answers {"ignored":true} while the file never changed is a lie the reader
	// has no way to notice until the book is synced again.
	return writeJSONFile(s.path, map[string]any{"items": s.items}, ".hardcover-ignored-*")
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
func (m *multiServer) setHardcoverIgnored(lib Library, book catalog.Book, ignore bool) error {
	if ignore {
		return m.hardcoverIgnored.add(ignoredBook{Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...), Reason: ignoredByYou, At: time.Now().Unix()})
	}
	// Syncing again may put the book back on the shelf, which is what resuming means.
	m.hardcoverMatches.forgetShelf(lib.ID, book.ID)
	return m.hardcoverIgnored.remove(lib.ID, book.ID)
}

// apiHardcoverIgnore serves /api/metadata/hardcover/ignored (the list) and /ignore (stop or resume syncing a book).
func (m *multiServer) apiHardcoverIgnore(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/ignored") {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		page := readHardcoverPage(r)
		all := m.hardcoverIgnored.list()
		filtered := make([]ignoredBook, 0, len(all))
		for _, item := range all {
			if hardcoverMatchesQuery(page, item.Title, item.Reason) {
				filtered = append(filtered, item)
			}
		}
		items, total := window(filtered, page)
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "offset": page.offset, "limit": page.limit})
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
	if err := m.setHardcoverIgnored(lib, book, input.Ignore); err != nil {
		http.Error(w, "Could not save the Hardcover setting", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ignored": input.Ignore})
}
