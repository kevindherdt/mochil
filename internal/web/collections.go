package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mochil/internal/catalog"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
	"mochil/internal/statefile"
)

type collectionMember struct {
	Type      string `json:"type"`
	LibraryID string `json:"library_id"`
	BookID    string `json:"book_id,omitempty"`
	Series    string `json:"series,omitempty"`
}

// collectionRule makes a collection "smart": it holds a search instead of picked books, so new books that match
// show up on their own. Empty fields are not filtered on.
type collectionRule struct {
	Query     string `json:"query,omitempty"`
	Author    string `json:"author,omitempty"`
	Series    string `json:"series,omitempty"`
	Status    string `json:"status,omitempty"`     // unread, reading or finished, for whoever owns the collection
	Format    string `json:"format,omitempty"`     // epub, pdf, cbz…
	Tag       string `json:"tag,omitempty"`        // one of the book's tags
	AddedDays int    `json:"added_days,omitempty"` // added or changed in the last this many days
	// MissingVolumes keeps only books of series that skip a whole-numbered volume, so the holes show up.
	MissingVolumes bool `json:"missing_volumes,omitempty"`
}

func (r collectionRule) empty() bool {
	return r.Query == "" && r.Author == "" && r.Series == "" && r.Status == "" && r.Format == "" && r.Tag == "" && r.AddedDays == 0 && !r.MissingVolumes
}

type savedCollection struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	LibraryID string             `json:"library_id"`
	Members   []collectionMember `json:"members"`
	Rule      *collectionRule    `json:"rule,omitempty"`
	CreatedAt int64              `json:"created_at"`
}
type collectionData struct {
	Version  int                          `json:"version"`
	Accounts map[string][]savedCollection `json:"accounts"`
}
type collectionStore struct {
	path string
	mu   sync.RWMutex
	data collectionData
}

const (
	maximumCollectionRequestBytes = 16 << 10
	maximumCollectionStateBytes   = 32 << 20
	// A collection detail can match thousands of books; the web UI asks for one
	// page at a time, the same size as the shelf. maximumCollectionPageSize is
	// the same ceiling /api/books uses, so no single write can exceed it.
	defaultCollectionPageSize = 120
	maximumCollectionPageSize = 500
)

// collectionPage reads limit/offset for a collection detail. A request without
// a limit keeps the original single-shot shape: every match, no window. A
// request that mentions limit is paged, with an unusable value falling back to
// the default page size rather than to the whole collection.
func collectionPage(query url.Values) (limit, offset int, paged bool) {
	if _, paged = query["limit"]; !paged {
		return 0, 0, false
	}
	limit = defaultCollectionPageSize
	if value, err := strconv.Atoi(query.Get("limit")); err == nil && value > 0 {
		limit = value
	}
	if limit > maximumCollectionPageSize {
		limit = maximumCollectionPageSize
	}
	if value, err := strconv.Atoi(query.Get("offset")); err == nil && value > 0 {
		offset = value
	}
	return limit, offset, true
}

func newCollectionStore(path string) (*collectionStore, error) {
	s := &collectionStore{path: path, data: collectionData{Version: 1, Accounts: map[string][]savedCollection{}}}
	if path != "" {
		if err := statefile.DecodeJSON(path, maximumCollectionStateBytes, &s.data); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	if s.data.Version != 1 {
		return nil, fmt.Errorf("unsupported collections version %d", s.data.Version)
	}
	if s.data.Accounts == nil {
		s.data.Accounts = map[string][]savedCollection{}
	}
	for account, collections := range s.data.Accounts {
		for i := range collections {
			if collections[i].LibraryID == "" && len(collections[i].Members) > 0 {
				libraryID := collections[i].Members[0].LibraryID
				for _, member := range collections[i].Members[1:] {
					if member.LibraryID != libraryID {
						libraryID = ""
						break
					}
				}
				collections[i].LibraryID = libraryID
			}
		}
		s.data.Accounts[account] = collections
	}
	return s, nil
}

// collectionSaveHook lets a test fail a chosen persist; nil in production.
var collectionSaveHook func() error

func (s *collectionStore) saveLocked() error {
	if collectionSaveHook != nil {
		if err := collectionSaveHook(); err != nil {
			return err
		}
	}
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".collections-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		e := json.NewEncoder(f)
		e.SetIndent("", "  ")
		err = e.Encode(s.data)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
func (s *collectionStore) persistLocked(previous collectionData) error {
	if err := s.saveLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}
func collectionID() string { var b [8]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }
func (s *collectionStore) list(account string) []savedCollection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]savedCollection(nil), s.data.Accounts[account]...)
}
func (s *collectionStore) deleteAccount(account string) error {
	if account == "" {
		return fmt.Errorf("invalid account")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.data.Accounts[account]; !exists {
		return nil
	}
	previous := cloneCollectionData(s.data)
	delete(s.data.Accounts, account)
	return s.persistLocked(previous)
}
func (s *collectionStore) create(account, name, libraryID string, rule *collectionRule) (savedCollection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneCollectionData(s.data)
	c := savedCollection{ID: collectionID(), Name: name, LibraryID: libraryID, Members: []collectionMember{}, Rule: rule, CreatedAt: time.Now().UnixMilli()}
	s.data.Accounts[account] = append(s.data.Accounts[account], c)
	return c, s.persistLocked(previous)
}

// createWithMembers creates a collection of picked books in ONE persist. Creating the empty collection and filling it in
// a second save left an empty twin behind whenever the second write failed: the caller answered 500, the name was free
// again, and the retry produced a duplicate.
func (m *multiServer) resolveCollection(account string, c savedCollection) []resolvedCollectionBook {
	if c.Rule != nil {
		return m.resolveSmartCollection(account, c)
	}
	seen := map[string]bool{}
	booksByLibrary := map[string][]catalog.Book{}
	var out []resolvedCollectionBook
	for _, member := range c.Members {
		lib, ok := m.library(member.LibraryID)
		if !ok || lib.Store == nil {
			continue
		}
		books, cached := booksByLibrary[lib.ID]
		if !cached {
			books = lib.Store.Books()
			booksByLibrary[lib.ID] = books
		}
		for _, book := range books {
			match := member.Type == "book" && book.ID == member.BookID || member.Type == "series" && book.Series == member.Series
			if !match {
				continue
			}
			key := lib.ID + "\x00" + book.ID
			if !seen[key] {
				seen[key] = true
				out = append(out, resolvedCollectionBook{lib, book})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.book.Series != b.book.Series {
			return catalog.NaturalLess(a.book.Series, b.book.Series)
		}
		if seriesLess(a.book, b.book) {
			return true
		}
		if seriesLess(b.book, a.book) {
			return false
		}
		if a.library.ID != b.library.ID {
			return a.library.ID < b.library.ID
		}
		return a.book.ID < b.book.ID
	})
	return out
}

// resolveSmartCollection runs the saved search now, so the collection is always current.
func (m *multiServer) resolveSmartCollection(account string, c savedCollection) []resolvedCollectionBook {
	if m.query == nil {
		return nil
	}
	request := catalogquery.BooksRequest{LibraryID: c.LibraryID, Query: c.Rule.Query, Limit: catalogquery.MaxPageSize}
	if c.Rule.Author != "" {
		request.Author = &c.Rule.Author
	}
	if c.Rule.Series != "" {
		request.Series = &c.Rule.Series
	}
	now := time.Now()
	var gaps map[string]bool
	if c.Rule.MissingVolumes {
		gaps = m.seriesWithGaps(c.LibraryID)
	}
	var out []resolvedCollectionBook
	for {
		page := m.query.Books(request)
		for _, item := range page.Items {
			lib, ok := m.library(item.LibraryID)
			if !ok {
				continue
			}
			var position reading.Position
			var hasPosition bool
			if c.Rule.needsReading() && m.reading != nil {
				position, hasPosition = m.reading.Get(account, item.LibraryID, item.Book.ID)
			}
			if gaps != nil && !gaps[gapKey(item.LibraryID, item.Book.Series)] {
				continue
			}
			if c.Rule.matchesDetails(item.Book, position, hasPosition, now) {
				out = append(out, resolvedCollectionBook{lib, item.Book})
			}
		}
		if !page.HasMore || len(page.Items) == 0 {
			return out
		}
		request.Offset += len(page.Items)
	}
}

func validMember(v collectionMember) bool {
	return (v.Type == "book" && v.LibraryID != "" && v.BookID != "" && v.Series == "") || (v.Type == "series" && v.LibraryID != "" && v.Series != "" && v.BookID == "")
}

func (m *multiServer) apiCollections(w http.ResponseWriter, r *http.Request) {
	if m.collections == nil {
		http.NotFound(w, r)
		return
	}
	account := m.readingAccount(r)
	if r.URL.Path == "/api/collections" {
		switch r.Method {
		case http.MethodGet:
			type summary struct {
				ID        string             `json:"id"`
				Name      string             `json:"name"`
				LibraryID string             `json:"library_id"`
				Count     int                `json:"count"`
				Members   []collectionMember `json:"members"`
				Rule      *collectionRule    `json:"rule,omitempty"`
				Covers    []string           `json:"covers"`
			}
			items := m.collections.list(account)
			libraryID := r.URL.Query().Get("library")
			out := make([]summary, 0, len(items))
			for _, c := range items {
				if libraryID != "" && c.LibraryID != libraryID {
					continue
				}
				books := m.resolveCollection(account, c)
				covers := []string{}
				for _, book := range books {
					if len(covers) == 3 {
						break
					}
					if url := m.bookView(book.library, book.book).CoverURL; url != "" {
						covers = append(covers, url)
					}
				}
				out = append(out, summary{c.ID, c.Name, c.LibraryID, len(books), c.Members, c.Rule, covers})
			}
			writeJSON(w, 200, out)
		case http.MethodPost:
			var in struct {
				Name      string          `json:"name"`
				LibraryID string          `json:"library_id"`
				Rule      *collectionRule `json:"rule"`
			}
			if decodeJSONRequest(w, r, maximumCollectionRequestBytes, &in, false) != nil || strings.TrimSpace(in.Name) == "" || len([]rune(strings.TrimSpace(in.Name))) > 80 || in.LibraryID == "" {
				http.Error(w, "Collection name is required", 400)
				return
			}
			library, ok := m.library(in.LibraryID)
			if !ok || library.Store == nil {
				http.Error(w, "Library not found", 400)
				return
			}
			if in.Rule != nil {
				rule := in.Rule.normalized()
				in.Rule = &rule
				if !rule.valid() {
					http.Error(w, "A smart collection needs a search", 400)
					return
				}
			}
			c, err := m.collections.create(account, strings.TrimSpace(in.Name), in.LibraryID, in.Rule)
			if err != nil {
				http.Error(w, "Collection could not be saved", 500)
				return
			}
			writeJSON(w, 201, c)
		default:
			http.Error(w, "Method not allowed", 405)
		}
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "books" && r.Method == http.MethodGet {
		for _, c := range m.collections.list(account) {
			if c.ID == id {
				resolved := m.resolveCollection(account, c)
				total := len(resolved)
				limit, offset, paged := collectionPage(r.URL.Query())
				if paged {
					if offset > total {
						offset = total
					}
					end := offset + limit
					if end > total {
						end = total
					}
					resolved = resolved[offset:end]
				}
				out := make([]bookView, 0, len(resolved))
				for _, v := range resolved {
					out = append(out, m.bookViewWithProgress(account, v.library, v.book))
				}
				// The page is a window over the whole match set; the header is
				// what tells the client whether there is another page.
				w.Header().Set("X-Total-Count", strconv.Itoa(total))
				writeJSON(w, 200, out)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "members" && (r.Method == http.MethodPost || r.Method == http.MethodDelete) {
		var member collectionMember
		if decodeJSONRequest(w, r, maximumCollectionRequestBytes, &member, false) != nil || !validMember(member) {
			http.Error(w, "Invalid collection member", 400)
			return
		}
		if err := m.collections.member(account, id, member, r.Method == http.MethodPost); err != nil {
			writeCollectionError(w, r, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var in struct {
			Name string          `json:"name"`
			Rule *collectionRule `json:"rule"`
		}
		if decodeJSONRequest(w, r, maximumCollectionRequestBytes, &in, false) != nil {
			http.Error(w, "Collection name is required", 400)
			return
		}
		name := strings.TrimSpace(in.Name)
		if (name == "" && in.Rule == nil) || len([]rune(name)) > 80 {
			http.Error(w, "Collection name is required", 400)
			return
		}
		c, err := m.collections.change(account, id, name, in.Rule)
		switch {
		case errors.Is(err, errNotSmartCollection), errors.Is(err, errInvalidRule):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case err != nil:
			// A failed write is a server fault, not a missing collection.
			writeCollectionError(w, r, err)
		default:
			writeJSON(w, 200, c)
		}
	case http.MethodDelete:
		if err := m.collections.remove(account, id); err != nil {
			writeCollectionError(w, r, err)
			return
		}
		w.WriteHeader(204)
	default:
		http.Error(w, "Method not allowed", 405)
	}
}

func (m *multiServer) opdsCollections(w http.ResponseWriter, r *http.Request) {
	account := m.readingAccount(r)
	rest := strings.TrimPrefix(r.URL.Path, "/opds/collections")
	items := m.collections.list(account)
	if rest == "" || rest == "/" {
		var all []catalog.Book
		libraryBooks := make(map[string][]catalog.Book, len(m.libraries))
		for _, l := range m.libraries {
			if l.Store == nil {
				continue
			}
			books := l.Store.Books()
			libraryBooks[l.ID] = books
			all = append(all, books...)
		}
		feed := m.root.feed("Collections", "urn:mochil:collections", all)
		libraryID := r.URL.Query().Get("library")
		if libraryID == "" {
			for _, library := range m.libraries {
				if library.Store == nil {
					continue
				}
				count := 0
				for _, c := range items {
					if c.LibraryID == library.ID {
						count++
					}
				}
				if count == 0 {
					continue
				}
				href := m.root.href("/opds/collections?library=" + url.QueryEscape(library.ID))
				feed.Entries = append(feed.Entries, navEntry(library.Name, fmt.Sprintf("%d collections", count), href, updated(libraryBooks[library.ID]), "collection-library:"+library.ID))
			}
			m.root.writeFeed(w, feed, "navigation")
			return
		}
		for _, c := range items {
			if c.LibraryID != libraryID {
				continue
			}
			href := m.root.href("/opds/collections/" + url.PathEscape(c.ID))
			feed.Entries = append(feed.Entries, navEntry(c.Name, fmt.Sprintf("%d books", len(m.resolveCollection(account, c))), href, updated(all), "collection:"+c.ID))
		}
		m.root.writeFeed(w, feed, "navigation")
		return
	}
	id, err := url.PathUnescape(strings.TrimPrefix(rest, "/"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	for _, c := range items {
		if c.ID != id {
			continue
		}
		resolved := m.resolveCollection(account, c)
		books := make([]catalog.Book, 0, len(resolved))
		for _, v := range resolved {
			b := v.book
			b.ID = v.library.ID + "/" + b.ID
			books = append(books, b)
		}
		feed := m.root.feed(c.Name, "urn:mochil:collection:"+c.ID, books)
		for _, v := range resolved {
			authors := make([]Person, 0, len(v.book.Authors))
			for _, a := range v.book.Authors {
				authors = append(authors, Person{Name: a})
			}
			name := filepath.Base(filepath.FromSlash(v.book.Path))
			href := m.root.href("/books/" + url.PathEscape(v.library.ID) + "/" + v.book.ID + "/" + url.PathEscape(name))
			feed.Entries = append(feed.Entries, Entry{ID: "urn:mochil:book:" + v.library.ID + ":" + v.book.ID, Title: v.book.Title, Updated: v.book.Modified.UTC().Format(time.RFC3339), Authors: authors, Content: Content{Type: "text", Text: v.book.Series}, Links: []Link{{Rel: acquisitionRel, Href: href, Type: v.book.MediaType, Title: "Download"}}})
		}
		m.root.writeFeed(w, feed, "acquisition")
		return
	}
	http.NotFound(w, r)
}
