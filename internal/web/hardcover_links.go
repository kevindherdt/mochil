package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mochil/internal/enrichment"
	"mochil/internal/reading"
	"mochil/internal/statefile"
)

// hardcoverLinkStore remembers which Hardcover book a library book is, for the ones the sync could not match by
// name. It is a plain list of pairs the reader chose, kept next to the other state files.
type hardcoverLinkStore struct {
	path string
	mu   sync.RWMutex
	ids  map[string]string // "library/book" -> Hardcover book id
}

func linkKey(libraryID, bookID string) string { return libraryID + "\x00" + bookID }

func openHardcoverLinks(path string) (*hardcoverLinkStore, error) {
	store := &hardcoverLinkStore{path: path, ids: map[string]string{}}
	if path == "" {
		return store, nil
	}
	var stored struct {
		Links map[string]string `json:"links"`
	}
	if err := statefile.DecodeJSON(path, 1<<20, &stored); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for key, id := range stored.Links {
		store.ids[key] = id
	}
	return store, nil
}

func (s *hardcoverLinkStore) get(libraryID, bookID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.ids[linkKey(libraryID, bookID)]
	return id, ok
}

// all returns every link as library, book and Hardcover id.
func (s *hardcoverLinkStore) all() [][3]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([][3]string, 0, len(s.ids))
	for key, id := range s.ids {
		library, book, _ := strings.Cut(key, "\x00")
		out = append(out, [3]string{library, book, id})
	}
	return out
}

func (s *hardcoverLinkStore) set(libraryID, bookID, hardcoverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, had := s.ids[linkKey(libraryID, bookID)]
	s.ids[linkKey(libraryID, bookID)] = hardcoverID
	if err := s.saveLocked(); err != nil {
		if had {
			s.ids[linkKey(libraryID, bookID)] = previous
		} else {
			delete(s.ids, linkKey(libraryID, bookID))
		}
		return err
	}
	return nil
}

func (s *hardcoverLinkStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	return writeJSONFile(s.path, map[string]any{"links": s.ids}, ".hardcover-links-*")
}

// writeJSONFile replaces a small state file in one step, so a crash never leaves half of it.
func writeJSONFile(path string, value any, tempPattern string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), tempPattern)
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(0o600)
	}
	// Renaming an unflushed file is how a crash turns a good state file into an
	// empty one, and these files are read at startup.
	if err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

// hardcoverMatch is what one library book was matched to on Hardcover, kept so the reader can check it.
type hardcoverMatch struct {
	Library          string   `json:"library"`
	Book             string   `json:"book"`
	Title            string   `json:"title"`
	Authors          []string `json:"authors"`
	HardcoverID      string   `json:"hardcover_id"`
	HardcoverTitle   string   `json:"hardcover_title,omitempty"`
	HardcoverAuthors []string `json:"hardcover_authors,omitempty"`
	URL              string   `json:"url,omitempty"`
	How              string   `json:"how"` // "matched" by name, or "linked" by hand
	Result           string   `json:"result"`
	// Doubt is set when the match deserves a second look, for instance when several Hardcover books fit.
	Doubt string `json:"doubt,omitempty"`
	At    int64  `json:"at"`
}

// A shelf is a reading history, so it grows without limit in principle. The cap is a guard against a runaway
// file, not a product limit: at 500 a reader with a few thousand books silently lost the oldest matches.
const maximumHardcoverMatches = 20000

// hardcoverMatches keeps the newest matches, newest first.
type hardcoverMatches struct {
	mu    sync.Mutex
	path  string
	items []hardcoverMatch
}

func (s *hardcoverMatches) open(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
	if path == "" {
		return nil
	}
	var stored struct {
		Items []hardcoverMatch `json:"items"`
	}
	if err := statefile.DecodeJSON(path, 4<<20, &stored); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.items = stored.Items
	return nil
}

func (s *hardcoverMatches) put(match hardcoverMatch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.items {
		if existing.Library == match.Library && existing.Book == match.Book {
			s.items = append(s.items[:i:i], s.items[i+1:]...)
			break
		}
	}
	s.items = append([]hardcoverMatch{match}, s.items...)
	if len(s.items) > maximumHardcoverMatches {
		s.items = s.items[:maximumHardcoverMatches]
	}
	if s.path != "" {
		_ = writeJSONFile(s.path, map[string]any{"items": s.items}, ".hardcover-matches-*")
	}
}

func (s *hardcoverMatches) get(libraryID, bookID string) (hardcoverMatch, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.items {
		if existing.Library == libraryID && existing.Book == bookID {
			return existing, true
		}
	}
	return hardcoverMatch{}, false
}

func (s *hardcoverMatches) list() []hardcoverMatch {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]hardcoverMatch{}, s.items...)
}

// hardcoverSkipped lists the books the sync could not match lately, so the reader can link them by hand.
type hardcoverSkipped struct {
	mu    sync.Mutex
	path  string // empty keeps the list in memory only
	items []skippedBook
}

type skippedBook struct {
	Library string   `json:"library"`
	Book    string   `json:"book"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	At      int64    `json:"at"`
}

const maximumSkippedBooks = 30

// open keeps the list in a file, so saving a setting (which restarts the server's workers) does not empty it.
func (s *hardcoverSkipped) open(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
	if path == "" {
		return nil
	}
	var stored struct {
		Items []skippedBook `json:"items"`
	}
	if err := statefile.DecodeJSON(path, 1<<20, &stored); err != nil && !os.IsNotExist(err) {
		return err
	}
	s.items = stored.Items
	if len(s.items) > maximumSkippedBooks {
		s.items = s.items[:maximumSkippedBooks]
	}
	return nil
}

func (s *hardcoverSkipped) saveLocked() {
	if s.path == "" {
		return
	}
	// Losing this list only costs the reader a book to read again, so a failed write is not worth failing the sync.
	_ = writeJSONFile(s.path, map[string]any{"items": s.items}, ".hardcover-skipped-*")
}

func (s *hardcoverSkipped) add(item skippedBook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeLocked(item.Library, item.Book)
	s.items = append([]skippedBook{item}, s.items...)
	if len(s.items) > maximumSkippedBooks {
		s.items = s.items[:maximumSkippedBooks]
	}
	s.saveLocked()
}

func (s *hardcoverSkipped) remove(libraryID, bookID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.items)
	s.removeLocked(libraryID, bookID)
	if len(s.items) != before {
		s.saveLocked()
	}
}

func (s *hardcoverSkipped) removeLocked(libraryID, bookID string) {
	for i, existing := range s.items {
		if existing.Library == libraryID && existing.Book == bookID {
			s.items = append(s.items[:i:i], s.items[i+1:]...)
			return
		}
	}
}

func (s *hardcoverSkipped) list() []skippedBook {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]skippedBook{}, s.items...)
}

// apiHardcover serves the "link a skipped book" screen: /api/metadata/hardcover/{skipped,search,link}. It lives under
// /api/metadata/, so only administrators reach it.
func (m *multiServer) apiHardcover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch strings.TrimPrefix(r.URL.Path, "/api/metadata/hardcover/") {
	case "skipped":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		page := readHardcoverPage(r)
		all := []skippedBook{}
		for _, item := range m.hardcoverSkipped.list() {
			if hardcoverMatchesQuery(page, item.Title, strings.Join(item.Authors, " ")) {
				all = append(all, item)
			}
		}
		items, total := window(all, page)
		writeJSON(w, http.StatusOK, map[string]any{"enabled": m.hardcoverEnqueue != nil, "items": items, "total": total, "offset": page.offset, "limit": page.limit})
	case "search":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" || len(query) > 300 {
			http.Error(w, "Enter a title to look for", http.StatusBadRequest)
			return
		}
		results, err := m.hardcover.Search(r.Context(), enrichment.Query{Title: query})
		if err != nil {
			status, message := providerConnectionError(err)
			http.Error(w, message, status)
			return
		}
		if results == nil {
			results = []enrichment.Suggestion{}
		}
		writeJSON(w, http.StatusOK, results)
	case "matches":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.recordLinkedMatches(r.Context(), "", "")
		page := readHardcoverPage(r)
		// "Check these" (a match worth a second look) and the rest are counted and paged apart, because the
		// page needs an honest total for each and never holds the whole shelf in memory.
		wantDoubt := r.URL.Query().Get("doubt")
		all := []hardcoverMatch{}
		for _, item := range m.hardcoverMatches.list() {
			if wantDoubt == "1" && item.Doubt == "" {
				continue
			}
			if wantDoubt == "0" && item.Doubt != "" {
				continue
			}
			if hardcoverMatchesQuery(page, item.Title, strings.Join(item.Authors, " "), item.HardcoverTitle) {
				all = append(all, item)
			}
		}
		items, total := window(all, page)
		writeJSON(w, http.StatusOK, map[string]any{"enabled": m.hardcoverEnqueue != nil, "items": items, "total": total, "offset": page.offset, "limit": page.limit})
	case "match":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		libraryID, bookID := r.URL.Query().Get("library"), r.URL.Query().Get("book")
		if _, _, ok := m.findBook(libraryID + "/" + bookID); !ok {
			http.NotFound(w, r)
			return
		}
		m.recordLinkedMatches(r.Context(), libraryID, bookID)
		out := map[string]any{"enabled": m.hardcoverEnqueue != nil, "match": nil, "skipped": false}
		if match, ok := m.hardcoverMatches.get(libraryID, bookID); ok {
			out["match"] = match
		}
		for _, item := range m.hardcoverSkipped.list() {
			if item.Library == libraryID && item.Book == bookID {
				out["skipped"] = true
			}
		}
		writeJSON(w, http.StatusOK, out)
	case "ignored", "ignore":
		m.apiHardcoverIgnore(w, r)
	case "preview":
		m.apiHardcoverPreview(w, r)
	case "link":
		m.apiHardcoverLink(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (m *multiServer) apiHardcoverLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Library    string `json:"library"`
		Book       string `json:"book"`
		ProviderID string `json:"provider_id"`
	}
	if err := decodeJSONRequest(w, r, 4<<10, &input, true); err != nil || m.hardcoverLinks == nil || m.hardcoverEnqueue == nil {
		http.Error(w, "Invalid Hardcover link", http.StatusBadRequest)
		return
	}
	for _, digit := range input.ProviderID {
		if digit < '0' || digit > '9' {
			http.Error(w, "Invalid Hardcover link", http.StatusBadRequest)
			return
		}
	}
	lib, book, ok := m.findBook(input.Library + "/" + input.Book)
	if input.ProviderID == "" || len(input.ProviderID) > 12 || !ok || !publicationKind(lib.Kind) {
		http.Error(w, "Invalid Hardcover link", http.StatusBadRequest)
		return
	}
	// Ask Hardcover what the chosen id is, so the choice can be shown back (and a typo in an id is caught).
	chosen, err := m.hardcover.BookByID(r.Context(), input.ProviderID)
	if errors.Is(err, enrichment.ErrHardcoverNoMatch) {
		http.Error(w, "Hardcover has no book with that id", http.StatusBadRequest)
		return
	}
	if err != nil {
		status, message := providerConnectionError(err)
		http.Error(w, message, status)
		return
	}
	if err := m.hardcoverLinks.set(lib.ID, book.ID, input.ProviderID); err != nil {
		http.Error(w, "The link could not be saved", http.StatusInternalServerError)
		return
	}
	m.hardcoverMatches.put(hardcoverMatch{
		Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...),
		HardcoverID: input.ProviderID, HardcoverTitle: chosen.Title, HardcoverAuthors: chosen.Authors, URL: chosen.SourceURL,
		How: "linked", Result: "linked, waiting to be sent", At: time.Now().Unix(),
	})
	m.hardcoverSkipped.remove(lib.ID, book.ID)
	// Send the book's current state now, so it does not wait for the next page turn.
	account := m.readingAccount(r)
	if position, found := m.reading.Get(account, lib.ID, book.ID); found && position.Fraction > 0 {
		event := reading.Reading
		if position.Finished {
			event = reading.Finished
		}
		m.hardcoverEnqueue(hardcoverJob{account: account, event: event, library: lib.ID, book: book.ID})
	}
	writeJSON(w, http.StatusOK, map[string]any{"linked": true, "url": "https://hardcover.app/books/" + url.PathEscape(input.ProviderID), "at": time.Now().Unix()})
}

// recordLinkedMatches gives a record to links made before records were kept (or by a Metadata suggestion), by asking
// Hardcover what each linked id is. With a library and book it handles only that one.
func (m *multiServer) recordLinkedMatches(ctx context.Context, onlyLibrary, onlyBook string) {
	if m.hardcoverLinks == nil || m.hardcoverEnqueue == nil {
		return
	}
	for _, link := range m.hardcoverLinks.all() {
		if onlyLibrary != "" && (link[0] != onlyLibrary || link[1] != onlyBook) {
			continue
		}
		if existing, ok := m.hardcoverMatches.get(link[0], link[1]); ok && existing.HardcoverID == link[2] {
			continue
		}
		lib, book, ok := m.findBook(link[0] + "/" + link[1])
		if !ok {
			continue
		}
		info := m.hardcoverBookInfo(ctx, lib.ID, book.ID, link[2])
		m.hardcoverMatches.put(hardcoverMatch{
			Library: lib.ID, Book: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...),
			HardcoverID: link[2], HardcoverTitle: info.Title, HardcoverAuthors: info.Authors, URL: info.SourceURL,
			How: "linked", Result: "linked", At: time.Now().Unix(),
		})
	}
}

// apiHardcoverPreview shows what a book would be matched to, and the closest alternatives, without sending anything.
func (m *multiServer) apiHardcoverPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, bookID := r.URL.Query().Get("library"), r.URL.Query().Get("book")
	lib, book, ok := m.findBook(libraryID + "/" + bookID)
	if !ok || !publicationKind(lib.Kind) {
		http.NotFound(w, r)
		return
	}
	out := map[string]any{"title": book.Title, "authors": book.Authors, "match": nil, "ambiguous": false, "linked": false, "candidates": []enrichment.Suggestion{}}
	if linked, ok := m.hardcoverLink(lib.ID, book.ID); ok {
		chosen := m.hardcoverBookInfo(r.Context(), lib.ID, book.ID, linked)
		out["match"], out["linked"] = chosen, true
		writeJSON(w, http.StatusOK, out)
		return
	}
	found, ambiguous, err := m.hardcover.Match(r.Context(), book.Title, book.Authors)
	switch {
	case err == nil:
		out["match"], out["ambiguous"] = found, ambiguous
	case errors.Is(err, enrichment.ErrHardcoverNoMatch):
		if candidates, searchErr := m.hardcover.Search(r.Context(), enrichment.Query{Title: book.Title}); searchErr == nil && candidates != nil {
			out["candidates"] = candidates
		}
	default:
		status, message := providerConnectionError(err)
		http.Error(w, message, status)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// forgetShelf drops the note that a book was put on the shelf, so a reader who asked to sync it again gets it added.
func (s *hardcoverMatches) forgetShelf(libraryID, bookID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.items {
		if existing.Library == libraryID && existing.Book == bookID {
			s.items[i].Result = "waiting to be sent"
			if s.path != "" {
				_ = writeJSONFile(s.path, map[string]any{"items": s.items}, ".hardcover-matches-*")
			}
			return
		}
	}
}
