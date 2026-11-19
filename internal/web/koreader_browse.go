package web

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
)

// The plugin lists a small page at a time: an e-reader on a slow link reads a screenful, not a catalogue.
const (
	koReaderBrowseDefaultLimit = 30
	koReaderBrowseMaxLimit     = 100
)

// koReaderBook is one book as the plugin lists it. Cover images are left out on purpose.
type koReaderBook struct {
	ID          string   `json:"id"`
	Library     string   `json:"library"`
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	Series      string   `json:"series,omitempty"`
	SeriesIndex *float64 `json:"series_index,omitempty"`
	Format      string   `json:"format"`
	ReadAlong   bool     `json:"read_along,omitempty"`
	Size        int64    `json:"size"`
	// Streamable and PageCount mirror the OPDS-PSE gating already used for the generic feed
	// (catalogue_http.go's bookView) and the Android app (androidPublicationPage): only a
	// library with PageStreaming enabled, a backend that implements PageBackend, and a book
	// with known comic pages can be streamed page-by-page via /api/koreader/v1/pages/....
	Streamable bool `json:"streamable,omitempty"`
	PageCount  int  `json:"page_count,omitempty"`
	// LastPage is the 1-indexed page the account last read, computed from the same fraction-based
	// progress-sync the plugin already pulls for "Continue reading" (reading.Store.Get). It is left
	// out when there is no recorded position, so the plugin opens a fresh stream at page 1.
	LastPage int `json:"last_page,omitempty"`
	// Finished mirrors reading.Position.Finished, so the plugin can offer a mark/unmark-finished
	// toggle from the catalog before ever downloading or opening the book.
	Finished bool `json:"finished,omitempty"`
}

// koReaderGroup is a series, author or collection the plugin can open.
type koReaderGroup struct {
	Name    string `json:"name"`
	Library string `json:"library,omitempty"`
	ID      string `json:"id,omitempty"`
	Count   int    `json:"count"`
	// For a library: how many series, authors and collections it holds, so the plugin can show them before opening.
	Series      int `json:"series,omitempty"`
	Authors     int `json:"authors,omitempty"`
	Collections int `json:"collections,omitempty"`
}

type koReaderBrowse struct {
	Kind   string          `json:"kind"`
	Total  int             `json:"total"`
	Books  []koReaderBook  `json:"books,omitempty"`
	Groups []koReaderGroup `json:"groups,omitempty"`
}

func koReaderBookOf(libraryID string, book catalog.Book) koReaderBook {
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(book.Path)), ".")
	if format == "" {
		format = strings.ToLower(book.Format)
	}
	authors := book.Authors
	if authors == nil {
		authors = []string{}
	}
	return koReaderBook{ID: book.ID, Library: libraryID, Title: book.Title, Authors: authors, Series: book.Series, SeriesIndex: book.SeriesIndex, Format: format, ReadAlong: book.ReadAlong, Size: book.Size}
}

// koReaderBookOf is the method form used by every production call site: it fills in the
// Streamable/PageCount/LastPage fields using the real Library (koReaderBookOf the free function
// has no access to Library.PageStreaming, the backend's PageBackend capability, or the account's
// reading position, which is why the test helper above calls it directly without those fields).
func (m *multiServer) koReaderBookOf(account, libraryID string, book catalog.Book) koReaderBook {
	out := koReaderBookOf(libraryID, book)
	var position reading.Position
	var hasPosition bool
	if m.reading != nil {
		position, hasPosition = m.reading.Get(account, libraryID, book.ID)
		out.Finished = hasPosition && position.Finished
	}
	if lib, ok := m.library(libraryID); ok && lib.PageStreaming && book.PageCount > 0 && book.PageMediaType != "" {
		if _, capable := catalog.PageCapability(lib.Store); capable {
			out.Streamable = true
			out.PageCount = book.PageCount
			if hasPosition && position.Fraction > 0 {
				page := int(position.Fraction*float64(book.PageCount)) + 1
				out.LastPage = min(max(page, 1), book.PageCount)
			}
		}
	}
	return out
}

func koReaderPage(r *http.Request) (offset, limit int) {
	limit = koReaderBrowseDefaultLimit
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 {
		limit = min(value, koReaderBrowseMaxLimit)
	}
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	return offset, limit
}

// koReaderPageBounds returns a safe slice window even when the requested offset
// is near MaxInt; adding the limit before clamping can overflow.
func koReaderPageBounds(offset, limit, total int) (start, end int) {
	return pageWindow(offset, limit, total)
}

// koReaderBrowseView answers one browse request. Series and author lists keep their natural order (series by volume).
func (m *multiServer) koReaderBrowseView(account string, r *http.Request) (koReaderBrowse, bool) {
	query := r.URL.Query()
	kind := query.Get("kind")
	search := strings.TrimSpace(query.Get("q"))
	offset, limit := koReaderPage(r)
	out := koReaderBrowse{Kind: kind}
	if kind != "" && kind != "books" && kind != "series" && kind != "authors" && kind != "collections" && kind != "libraries" {
		return out, false
	}
	if m.query == nil {
		return out, true
	}
	selectedSeries, selectedAuthor, collection := query.Has("series"), query.Has("author"), query.Get("collection")
	libraryFilter := query.Get("library")
	switch {
	case kind == "libraries":
		groups := m.koReaderLibraries(account, search)
		out.Total = len(groups)
		start, end := koReaderPageBounds(offset, limit, len(groups))
		out.Groups = groups[start:end]
		return out, true
	case collection != "":
		for _, saved := range m.collections.list(account) {
			if saved.ID != collection {
				continue
			}
			resolved := m.resolveCollection(account, saved)
			if search != "" {
				needle := strings.ToLower(search)
				filtered := resolved[:0]
				for _, entry := range resolved {
					book := entry.book
					key := strings.ToLower(book.Title + " " + book.Series + " " + strings.Join(book.Authors, " "))
					if strings.Contains(key, needle) {
						filtered = append(filtered, entry)
					}
				}
				resolved = filtered
			}
			out.Kind, out.Total = "books", len(resolved)
			start, end := koReaderPageBounds(offset, limit, len(resolved))
			for _, entry := range resolved[start:end] {
				out.Books = append(out.Books, m.koReaderBookOf(account, entry.library.ID, entry.book))
			}
			return out, true
		}
		return out, false
	case kind == "collections":
		for _, saved := range m.collections.list(account) {
			if libraryFilter != "" && saved.LibraryID != libraryFilter {
				continue
			}
			if search == "" || strings.Contains(strings.ToLower(saved.Name), strings.ToLower(search)) {
				out.Groups = append(out.Groups, koReaderGroup{Name: saved.Name, ID: saved.ID, Library: saved.LibraryID, Count: len(m.resolveCollection(account, saved))})
			}
		}
		out.Total = len(out.Groups)
		start, end := koReaderPageBounds(offset, limit, len(out.Groups))
		out.Groups = out.Groups[start:end]
		return out, true
	case (kind == "series" && !selectedSeries) || (kind == "authors" && !selectedAuthor):
		groups := m.koReaderGroups(kind, search, libraryFilter)
		out.Total = len(groups)
		start, end := koReaderPageBounds(offset, limit, len(groups))
		out.Groups = groups[start:end]
		return out, true
	}
	request := catalogquery.BooksRequest{LibraryID: query.Get("library"), Query: search, Limit: limit, Offset: offset}
	if selectedSeries {
		name := query.Get("series")
		request.Series = &name
		// Sort all matching volumes before the query applies pagination.
		request.Sort = "series_volume"
	}
	if selectedAuthor {
		name := query.Get("author")
		request.Author = &name
	}
	result := m.query.Books(request)
	out.Kind, out.Total = "books", result.Total
	for _, item := range result.Items {
		out.Books = append(out.Books, m.koReaderBookOf(account, item.LibraryID, item.Book))
	}
	return out, true
}

// koReaderGroups lists the series or authors the plugin can open. "Standalone books" and "Unknown author" are
// catch-alls rather than real groups, so they are left out, and the home page counts the same list.
//
// Fetches everything in one unlimited (Limit<=0) call instead of paging through MaxPageSize chunks: the query
// layer already treats Limit<=0 as "return every item" (paginateSeries/paginateAuthors and the per-library
// seriesPage/authorPage fast paths in internal/query all honor this), so one call does the same work the old
// chunked loop did across several -- except a search query re-scans the library per call, so the old loop paid
// for that rescan once per MaxPageSize-sized chunk of matching groups instead of once.
func (m *multiServer) koReaderGroups(kind, search, libraryID string) []koReaderGroup {
	request := catalogquery.SeriesRequest{LibraryID: libraryID, Query: search}
	var groups []koReaderGroup
	if kind == "series" {
		for _, item := range m.query.Series(request).Items {
			if !item.Unsorted {
				groups = append(groups, koReaderGroup{Name: item.Name, Library: item.LibraryID, Count: item.Count})
			}
		}
	} else {
		for _, item := range m.query.Authors(request).Items {
			if !item.Unknown {
				groups = append(groups, koReaderGroup{Name: item.Name, Library: item.LibraryID, Count: item.Count})
			}
		}
	}
	return groups
}

// koReaderGroupCount is koReaderGroups' count-only sibling: the library picker only needs how many series or
// authors a library has, not their names, so this skips building a koReaderGroup per entry.
func (m *multiServer) koReaderGroupCount(kind, libraryID string) int {
	request := catalogquery.SeriesRequest{LibraryID: libraryID}
	count := 0
	if kind == "series" {
		for _, item := range m.query.Series(request).Items {
			if !item.Unsorted {
				count++
			}
		}
	} else {
		for _, item := range m.query.Authors(request).Items {
			if !item.Unknown {
				count++
			}
		}
	}
	return count
}

// koReaderLibraries lists the libraries that hold books, with what is inside each, so the plugin can offer
// "choose a library, then browse it" instead of one pile of everything.
func (m *multiServer) koReaderLibraries(account, search string) []koReaderGroup {
	var groups []koReaderGroup
	for _, library := range m.libraries {
		if !publicationKind(library.Kind) || library.Store == nil || !strings.Contains(strings.ToLower(library.Name), strings.ToLower(search)) {
			continue
		}
		group := koReaderGroup{
			Name: library.Name, Library: library.ID, ID: library.ID,
			Count:  m.query.Books(catalogquery.BooksRequest{LibraryID: library.ID, Limit: 1}).Total,
			Series: m.koReaderGroupCount("series", library.ID), Authors: m.koReaderGroupCount("authors", library.ID),
		}
		if m.collections != nil {
			for _, saved := range m.collections.list(account) {
				if saved.LibraryID == library.ID {
					group.Collections++
				}
			}
		}
		groups = append(groups, group)
	}
	return groups
}

func (m *multiServer) serveKoReaderBrowse(w http.ResponseWriter, r *http.Request, account string) {
	result, ok := m.koReaderBrowseView(account, r)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown_view"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

// serveKoReaderDownload sends one book file to a paired device: /api/koreader/v1/download/{library}/{id}.
func (m *multiServer) serveKoReaderDownload(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/download/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	libraryID, err1 := url.PathUnescape(parts[0])
	bookID, err2 := url.PathUnescape(parts[1])
	library, found := m.library(libraryID)
	if err1 != nil || err2 != nil || !found || library.Store == nil {
		http.NotFound(w, r)
		return
	}
	book, found := library.Store.Book(bookID)
	if !found {
		http.NotFound(w, r)
		return
	}
	acquisition, err := library.Store.Acquire(r.Context(), book)
	if err != nil || acquisition.Reader == nil {
		http.Error(w, "Book is temporarily unavailable", http.StatusBadGateway)
		return
	}
	defer acquisition.Reader.Close()
	mediaType := book.MediaType
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filepath.Base(book.Path)))
	http.ServeContent(w, r, filepath.Base(book.Path), book.Modified, acquisition.Reader)
}

// serveKoReaderPage serves one OPDS-PSE comic page over the Bearer-gated KOReader API:
// /api/koreader/v1/pages/{library}/{id}/{page}. It mirrors android_devices.go's
// androidPublicationPage (itself mirroring web.go's session/Basic-only streamPage) so the
// Mochil KOReader plugin's lazy page table (core.lua, modeled on upstream
// plugins/opds.koplugin/opdspse.lua) can fetch pages with its device Bearer token, which
// the generic session/Basic-gated /stream/{id} route never accepts.
func (m *multiServer) serveKoReaderPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	remainder := strings.TrimPrefix(r.URL.Path, "/api/koreader/v1/pages/")
	bookPath, pageSegment, ok := cutLastSlash(remainder)
	if !ok {
		http.NotFound(w, r)
		return
	}
	library, book, ok := m.findBook(bookPath)
	if !ok {
		http.NotFound(w, r)
		return
	}
	backend, ok := catalog.PageCapability(library.Store)
	if !library.PageStreaming || !ok {
		http.NotFound(w, r)
		return
	}
	index, err := strconv.Atoi(pageSegment)
	if err != nil || index < 0 || index >= book.PageCount {
		http.Error(w, "Invalid page", http.StatusBadRequest)
		return
	}
	page, err := backend.Page(r.Context(), book, index)
	if err != nil {
		http.Error(w, "Page is temporarily unavailable", http.StatusBadGateway)
		return
	}
	defer page.Reader.Close()
	if page.MediaType == "" {
		http.Error(w, "Unsupported page image", http.StatusUnsupportedMediaType)
		return
	}
	w.Header().Set("Content-Type", page.MediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if page.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(page.Size, 10))
	}
	_, _ = io.Copy(w, page.Reader)
}

// cutLastSlash splits "{library}/{id}/{page}" into ("{library}/{id}", "{page}") at the final
// slash, since findBook expects the library/id pair and library or book IDs cannot themselves
// contain an unescaped slash.
func cutLastSlash(remainder string) (bookPath, last string, ok bool) {
	index := strings.LastIndex(remainder, "/")
	if index < 0 {
		return "", "", false
	}
	return remainder[:index], remainder[index+1:], true
}
