package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"mochil/internal/statefile"
)

const overrideFileVersion = 1
const maximumOverrideStateBytes int64 = 64 << 20

type MetadataOverride struct {
	Title       string            `json:"title,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Series      string            `json:"series,omitempty"`
	SeriesIndex *float64          `json:"series_index,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Publisher   string            `json:"publisher,omitempty"`
	Published   string            `json:"published,omitempty"`
	Language    string            `json:"language,omitempty"`
	Rating      *int              `json:"rating,omitempty"` // 0 to 10, as Calibre stores it
	Description string            `json:"description,omitempty"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
	TitleSort   string            `json:"title_sort,omitempty"`
	AuthorSort  string            `json:"author_sort,omitempty"`
	ArtworkURL  string            `json:"artwork_url,omitempty"`
	Provider    string            `json:"provider,omitempty"`
	ProviderID  string            `json:"provider_id,omitempty"`
}

func (s *OverrideStore) Get(libraryID, bookID string) (MetadataOverride, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data.Libraries[libraryID][bookID]
	value.Authors = append([]string(nil), value.Authors...)
	return value, ok
}

// HasOverrides reports whether the library has any correction at all, so a
// caller can hand out the published catalogue untouched when it has none.
func (s *OverrideStore) HasOverrides(libraryID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Libraries[libraryID]) > 0
}

func (s *OverrideStore) SetArtwork(libraryID, bookID, artworkURL, provider, providerID string) error {
	if strings.TrimSpace(libraryID) == "" || strings.TrimSpace(bookID) == "" || strings.TrimSpace(artworkURL) == "" || strings.TrimSpace(provider) == "" {
		return errors.New("artwork override is incomplete")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	books := s.data.Libraries[libraryID]
	rollbackLibrary := func() {}
	if books == nil {
		books = map[string]MetadataOverride{}
		s.data.Libraries[libraryID] = books
		rollbackLibrary = func() { delete(s.data.Libraries, libraryID) }
	}
	priorValue, hadValue := books[bookID]
	value := priorValue
	value.ArtworkURL = strings.TrimSpace(artworkURL)
	value.Provider = strings.TrimSpace(provider)
	value.ProviderID = strings.TrimSpace(providerID)
	books[bookID] = value
	if err := s.saveLocked(); err != nil {
		if hadValue {
			books[bookID] = priorValue
		} else {
			delete(books, bookID)
		}
		rollbackLibrary()
		return err
	}
	s.generation++
	return nil
}

func (s *OverrideStore) DeleteArtwork(libraryID, bookID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	books := s.data.Libraries[libraryID]
	priorValue, ok := books[bookID]
	if !ok || priorValue.ArtworkURL == "" {
		return nil
	}
	value := priorValue
	value.ArtworkURL, value.Provider, value.ProviderID = "", "", ""
	books[bookID] = value
	if err := s.saveLocked(); err != nil {
		books[bookID] = priorValue
		return err
	}
	s.generation++
	return nil
}

type overrideFile struct {
	Version   int                                    `json:"version"`
	Libraries map[string]map[string]MetadataOverride `json:"libraries"`
}

// OverrideStore owns explicit, reversible catalogue corrections for sources
// that Mochil must not modify. It never changes canonical media.
type OverrideStore struct {
	path       string
	mu         sync.RWMutex
	data       overrideFile
	generation uint64
}

func OpenOverrideStore(path string) (*OverrideStore, error) {
	store := &OverrideStore{path: path, data: overrideFile{Version: overrideFileVersion, Libraries: map[string]map[string]MetadataOverride{}}}
	if path == "" {
		return store, nil
	}
	err := statefile.DecodeJSON(path, maximumOverrideStateBytes, &store.data)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("decode metadata overrides: %w", err)
	}
	if store.data.Version != overrideFileVersion {
		return nil, fmt.Errorf("unsupported metadata override version %d", store.data.Version)
	}
	if store.data.Libraries == nil {
		store.data.Libraries = map[string]map[string]MetadataOverride{}
	}
	return store, nil
}

func (s *OverrideStore) Apply(libraryID string, changes map[string]map[string]string) error {
	_, err := s.ApplyWithUndo(libraryID, changes)
	return err
}

// ApplyWithUndo persists a complete batch and returns a compensating action for
// a larger transaction, such as a metadata review job spanning source writes.
func (s *OverrideStore) ApplyWithUndo(libraryID string, changes map[string]map[string]string) (func() error, error) {
	if strings.TrimSpace(libraryID) == "" || len(changes) == 0 {
		return nil, errors.New("metadata overrides require a library and at least one change")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	books := s.data.Libraries[libraryID]
	rollbackLibrary := func() {}
	if books == nil {
		books = map[string]MetadataOverride{}
		s.data.Libraries[libraryID] = books
		rollbackLibrary = func() { delete(s.data.Libraries, libraryID) }
	}
	// Capture only the entries this batch is about to touch (bounded by
	// len(changes)), not a clone of every library's every override, so a
	// mid-batch validation failure or a failed persist can be undone
	// without copying anyone else's data.
	type priorEntry struct {
		value MetadataOverride
		had   bool
	}
	captured := map[string]priorEntry{}
	capture := func(bookID string) {
		if _, done := captured[bookID]; done {
			return
		}
		value, had := books[bookID]
		captured[bookID] = priorEntry{value, had}
	}
	abort := func() {
		for bookID, entry := range captured {
			if entry.had {
				books[bookID] = entry.value
			} else {
				delete(books, bookID)
			}
		}
		rollbackLibrary()
	}
	for bookID, fields := range changes {
		if strings.TrimSpace(bookID) == "" {
			abort()
			return nil, errors.New("metadata override book ID is empty")
		}
		capture(bookID)
		value := books[bookID]
		for field, raw := range fields {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				abort()
				return nil, fmt.Errorf("metadata override %s cannot be empty", field)
			}
			switch field {
			case "title":
				value.Title = raw
			case "authors":
				value.Authors = splitOverrideAuthors(raw)
			case "series":
				value.Series = raw
			case "tags":
				value.Tags = splitOverrideList(raw)
			case "publisher":
				value.Publisher = raw
			case "published":
				value.Published = raw
			case "language":
				value.Language = raw
			case "description":
				value.Description = raw
			case "title_sort":
				value.TitleSort = raw
			case "author_sort":
				value.AuthorSort = raw
			case "rating":
				stars, err := strconv.ParseFloat(raw, 64)
				if err != nil || stars < 0 || stars > 5 || math.IsNaN(stars) {
					abort()
					return nil, errors.New("metadata override rating must be between 0 and 5 stars")
				}
				rating := int(math.Round(stars * 2))
				value.Rating = &rating
			case "identifiers":
				identifiers, err := parseOverrideIdentifiers(raw)
				if err != nil {
					abort()
					return nil, err
				}
				value.Identifiers = identifiers
			case "series_index":
				number, err := strconv.ParseFloat(raw, 64)
				if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
					abort()
					return nil, errors.New("metadata override volume number is invalid")
				}
				value.SeriesIndex = &number
			default:
				abort()
				return nil, fmt.Errorf("unsupported metadata override field %q", field)
			}
		}
		books[bookID] = value
	}
	if err := s.saveLocked(); err != nil {
		abort()
		return nil, err
	}
	s.generation++
	undo := func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		currentBooks := s.data.Libraries[libraryID]
		rollbackUndoLibrary := func() {}
		if currentBooks == nil {
			currentBooks = map[string]MetadataOverride{}
			s.data.Libraries[libraryID] = currentBooks
			rollbackUndoLibrary = func() { delete(s.data.Libraries, libraryID) }
		}
		// Capture the state right before undo runs so a failed persist
		// here restores exactly that, not the pre-apply state.
		beforeUndo := map[string]priorEntry{}
		for bookID, entry := range captured {
			value, had := currentBooks[bookID]
			beforeUndo[bookID] = priorEntry{value, had}
			if entry.had {
				currentBooks[bookID] = entry.value
			} else {
				delete(currentBooks, bookID)
			}
		}
		if err := s.saveLocked(); err != nil {
			for bookID, entry := range beforeUndo {
				if entry.had {
					currentBooks[bookID] = entry.value
				} else {
					delete(currentBooks, bookID)
				}
			}
			rollbackUndoLibrary()
			return err
		}
		s.generation++
		return nil
	}
	return undo, nil
}

// Reset forgets the named edits of one book (title, authors, series, series_index), so it reads from its files
// again. No fields means every edit of the book except its artwork. It returns an undo like ApplyWithUndo.
func (s *OverrideStore) Reset(libraryID, bookID string, fields ...string) (func() error, error) {
	for _, field := range fields {
		switch field {
		case "title", "authors", "series", "series_index", "tags", "publisher", "published", "language", "rating", "description", "identifiers", "title_sort", "author_sort":
		default:
			return nil, fmt.Errorf("unsupported metadata override field %q", field)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	books := s.data.Libraries[libraryID]
	priorValue, ok := books[bookID]
	if !ok {
		return func() error { return nil }, nil
	}
	all := len(fields) == 0
	has := func(name string) bool {
		for _, field := range fields {
			if field == name {
				return true
			}
		}
		return all
	}
	value := priorValue
	if has("title") {
		value.Title = ""
	}
	if has("authors") {
		value.Authors = nil
	}
	if has("series") {
		value.Series = ""
	}
	if has("series_index") {
		value.SeriesIndex = nil
	}
	if has("tags") {
		value.Tags = nil
	}
	if has("publisher") {
		value.Publisher = ""
	}
	if has("published") {
		value.Published = ""
	}
	if has("language") {
		value.Language = ""
	}
	if has("rating") {
		value.Rating = nil
	}
	if has("description") {
		value.Description = ""
	}
	if has("identifiers") {
		value.Identifiers = nil
	}
	if has("title_sort") {
		value.TitleSort = ""
	}
	if has("author_sort") {
		value.AuthorSort = ""
	}
	if value.Title == "" && len(value.Authors) == 0 && value.Series == "" && value.SeriesIndex == nil && value.ArtworkURL == "" && value.Provider == "" && value.ProviderID == "" &&
		len(value.Tags) == 0 && value.Publisher == "" && value.Published == "" && value.Language == "" && value.Rating == nil && value.Description == "" && len(value.Identifiers) == 0 && value.TitleSort == "" && value.AuthorSort == "" {
		delete(books, bookID)
	} else {
		books[bookID] = value
	}
	if err := s.saveLocked(); err != nil {
		books[bookID] = priorValue
		return nil, err
	}
	s.generation++
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		currentBooks := s.data.Libraries[libraryID]
		rollbackUndoLibrary := func() {}
		if currentBooks == nil {
			currentBooks = map[string]MetadataOverride{}
			s.data.Libraries[libraryID] = currentBooks
			rollbackUndoLibrary = func() { delete(s.data.Libraries, libraryID) }
		}
		beforeUndo, hadBeforeUndo := currentBooks[bookID]
		currentBooks[bookID] = priorValue
		if err := s.saveLocked(); err != nil {
			if hadBeforeUndo {
				currentBooks[bookID] = beforeUndo
			} else {
				delete(currentBooks, bookID)
			}
			rollbackUndoLibrary()
			return err
		}
		s.generation++
		return nil
	}, nil
}

func (s *OverrideStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".metadata-overrides-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s.data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
