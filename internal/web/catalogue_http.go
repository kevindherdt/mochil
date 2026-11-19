package web

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"mochil/internal/catalog"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
)

type libraryView struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Count            int    `json:"count"`
	MetadataProvider string `json:"metadata_provider,omitempty"`
}

type bookView struct {
	catalog.Book
	LibraryID     string                   `json:"library_id"`
	LibraryName   string                   `json:"library_name"`
	LibraryKind   string                   `json:"library_kind"`
	CoverURL      string                   `json:"cover_url"`
	ReadURL       string                   `json:"read_url"`
	DownloadURL   string                   `json:"download_url"`
	PageStreamURL string                   `json:"page_stream_url,omitempty"`
	Bucket        string                   `json:"bucket"`
	Reading       *bookReadingProgressView `json:"reading_progress,omitempty"`
}

type bookReadingProgressView struct {
	Fraction  float64 `json:"fraction"`
	UpdatedAt int64   `json:"updated_at"`
	Finished  bool    `json:"finished,omitempty"`
	Status    string  `json:"status,omitempty"`
}

type bookNavigationView struct {
	Bucket string `json:"bucket"`
	Offset int    `json:"offset"`
	Count  int    `json:"count"`
}

type seriesView struct {
	LibraryID   string   `json:"library_id"`
	LibraryName string   `json:"library_name"`
	Name        string   `json:"name"`
	Authors     []string `json:"authors,omitempty"`
	Bucket      string   `json:"bucket"`
	Count       int      `json:"count"`
	Unsorted    bool     `json:"unsorted,omitempty"`
	CoverURL    string   `json:"cover_url,omitempty"`
}

type authorView struct {
	LibraryID   string `json:"library_id"`
	LibraryName string `json:"library_name"`
	Name        string `json:"name"`
	Bucket      string `json:"bucket"`
	Count       int    `json:"count"`
	Unknown     bool   `json:"unknown,omitempty"`
	CoverURL    string `json:"cover_url,omitempty"`
}

func (m *multiServer) apiLibraries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	results := m.query.Libraries()
	out := make([]libraryView, 0, len(m.libraries))
	for _, library := range results {
		configured, _ := m.library(library.ID)
		out = append(out, libraryView{ID: library.ID, Name: library.Name, Kind: library.Kind, Count: library.Count, MetadataProvider: configured.MetadataProvider})
	}
	for _, library := range m.libraries {
		if library.Video != nil {
			out = append(out, libraryView{ID: library.ID, Name: library.Name, Kind: library.Kind, Count: library.Video.Count(), MetadataProvider: library.MetadataProvider})
		} else if library.Audio != nil {
			out = append(out, libraryView{ID: library.ID, Name: library.Name, Kind: library.Kind, Count: library.Audio.Count(), MetadataProvider: library.MetadataProvider})
		} else if library.Documents != nil {
			out = append(out, libraryView{ID: library.ID, Name: library.Name, Kind: library.Kind, Count: library.Documents.Count(), MetadataProvider: library.MetadataProvider})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	wantLibrary := r.URL.Query().Get("library")
	wantSeries, hasSeries := r.URL.Query()["series"]
	wantAuthor, hasAuthor := r.URL.Query()["author"]
	if hasSeries && hasAuthor {
		http.Error(w, "Book group filter is invalid", http.StatusBadRequest)
		return
	}
	queryValue, bucketValue, sortValue := r.URL.Query().Get("q"), r.URL.Query().Get("bucket"), strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortValue != "" && sortValue != "recent" && sortValue != "author" && sortValue != "series" {
		http.Error(w, "Book sort is invalid", http.StatusBadRequest)
		return
	}
	seriesValue := ""
	if hasSeries {
		seriesValue = wantSeries[0]
	}
	authorValue := ""
	if hasAuthor {
		authorValue = wantAuthor[0]
	}
	view := publicationView("books", wantLibrary, strings.TrimSpace(queryValue), strings.ToUpper(strings.TrimSpace(bucketValue)), strconv.FormatBool(hasSeries), seriesValue, strconv.FormatBool(hasAuthor), authorValue, sortValue)
	cursor, cursorErr := decodePublicationCursor(r.URL.Query().Get("cursor"))
	if cursorErr != nil || (cursor.Kind != "" && (cursor.Kind != "books" || cursor.View != view)) {
		http.Error(w, "Book cursor is invalid for this view", http.StatusBadRequest)
		return
	}
	version := m.query.Version(wantLibrary)
	if cursor.Kind != "" && cursor.Version != version {
		http.Error(w, "Book cursor expired; reload the shelf", http.StatusConflict)
		return
	}
	limit := 120
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= catalogquery.MaxPageSize {
		limit = value
	}
	offset := 0
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	request := catalogquery.BooksRequest{LibraryID: wantLibrary, Query: queryValue, Bucket: bucketValue, Limit: limit, Offset: offset, AfterLibraryID: cursor.LibraryID, AfterBookID: cursor.ItemID, Sort: sortValue}
	if hasSeries {
		request.Series = &wantSeries[0]
	}
	if hasAuthor {
		request.Author = &wantAuthor[0]
	}
	result := m.query.Books(request)
	if cursor.Kind != "" && !result.AnchorFound || version != m.query.Version(wantLibrary) {
		http.Error(w, "Book cursor expired; reload the shelf", http.StatusConflict)
		return
	}
	out := make([]bookView, 0, len(result.Items))
	account := m.readingAccount(r)
	for _, item := range result.Items {
		if library, ok := m.library(item.LibraryID); ok {
			out = append(out, m.bookViewWithProgress(account, library, item.Book))
		}
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(result.Total))
	if result.HasMore && len(result.Items) > 0 {
		last := result.Items[len(result.Items)-1]
		w.Header().Set("X-Next-Cursor", encodePublicationCursor(publicationCursor{Kind: "books", Version: version, View: view, LibraryID: last.LibraryID, ItemID: last.Book.ID}))
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiAuthors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, queryValue, bucketValue := r.URL.Query().Get("library"), r.URL.Query().Get("q"), r.URL.Query().Get("bucket")
	view := publicationView("authors", libraryID, strings.TrimSpace(queryValue), strings.ToUpper(strings.TrimSpace(bucketValue)))
	cursor, cursorErr := decodePublicationCursor(r.URL.Query().Get("cursor"))
	if cursorErr != nil || cursor.Kind != "" && (cursor.Kind != "authors" || cursor.View != view) {
		http.Error(w, "Author cursor is invalid for this view", http.StatusBadRequest)
		return
	}
	version := m.query.Version(libraryID)
	if cursor.Kind != "" && cursor.Version != version {
		http.Error(w, "Author cursor expired; reload the index", http.StatusConflict)
		return
	}
	limit, offset := 0, 0
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= catalogquery.MaxPageSize {
		limit = value
	}
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	page := m.query.Authors(catalogquery.SeriesRequest{LibraryID: libraryID, Query: queryValue, Bucket: bucketValue, Limit: limit, Offset: offset, AfterLibraryID: cursor.LibraryID, AfterID: cursor.ItemID})
	if cursor.Kind != "" && !page.AnchorFound || version != m.query.Version(libraryID) {
		http.Error(w, "Author cursor expired; reload the index", http.StatusConflict)
		return
	}
	out := make([]authorView, 0, len(page.Items))
	for _, result := range page.Items {
		coverURL := ""
		if library, ok := m.library(result.LibraryID); ok {
			coverURL = m.bookView(library, result.Representative).CoverURL
		}
		out = append(out, authorView{LibraryID: result.LibraryID, LibraryName: result.LibraryName, Name: result.Name, Bucket: result.Bucket, Count: result.Count, Unknown: result.Unknown, CoverURL: coverURL})
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(page.Total))
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		w.Header().Set("X-Next-Cursor", encodePublicationCursor(publicationCursor{Kind: "authors", Version: version, View: view, LibraryID: last.LibraryID, ItemID: last.Name}))
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiBookNavigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	results := m.query.BookNavigation(r.URL.Query().Get("library"), r.URL.Query().Get("q"))
	out := make([]bookNavigationView, 0, len(results))
	for _, result := range results {
		out = append(out, bookNavigationView{Bucket: result.Bucket, Offset: result.Offset, Count: result.Count})
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	libraryID, queryValue, bucketValue := r.URL.Query().Get("library"), r.URL.Query().Get("q"), r.URL.Query().Get("bucket")
	view := publicationView("series", libraryID, strings.TrimSpace(queryValue), strings.ToUpper(strings.TrimSpace(bucketValue)))
	cursor, cursorErr := decodePublicationCursor(r.URL.Query().Get("cursor"))
	if cursorErr != nil || (cursor.Kind != "" && (cursor.Kind != "series" || cursor.View != view)) {
		http.Error(w, "Series cursor is invalid for this view", http.StatusBadRequest)
		return
	}
	version := m.query.Version(libraryID)
	if cursor.Kind != "" && cursor.Version != version {
		http.Error(w, "Series cursor expired; reload the index", http.StatusConflict)
		return
	}
	limit, offset := 0, 0
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= catalogquery.MaxPageSize {
		limit = value
	}
	if value, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && value > 0 {
		offset = value
	}
	page := m.query.Series(catalogquery.SeriesRequest{LibraryID: libraryID, Query: queryValue, Bucket: bucketValue, Limit: limit, Offset: offset, AfterLibraryID: cursor.LibraryID, AfterID: cursor.ItemID})
	if cursor.Kind != "" && !page.AnchorFound || version != m.query.Version(libraryID) {
		http.Error(w, "Series cursor expired; reload the index", http.StatusConflict)
		return
	}
	out := make([]seriesView, 0, len(page.Items))
	for _, result := range page.Items {
		coverURL := ""
		if library, ok := m.library(result.LibraryID); ok {
			coverURL = m.bookView(library, result.Representative).CoverURL
		}
		out = append(out, seriesView{LibraryID: result.LibraryID, LibraryName: result.LibraryName, Name: result.Name, Authors: result.Representative.Authors, Bucket: result.Bucket, Count: result.Count, Unsorted: result.Unsorted, CoverURL: coverURL})
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(page.Total))
	if page.HasMore && len(page.Items) > 0 {
		last := page.Items[len(page.Items)-1]
		w.Header().Set("X-Next-Cursor", encodePublicationCursor(publicationCursor{Kind: "series", Version: version, View: view, LibraryID: last.LibraryID, ItemID: last.Representative.ID}))
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *multiServer) apiBook(w http.ResponseWriter, r *http.Request) {
	lib, book, ok := m.findBook(strings.TrimPrefix(r.URL.Path, "/api/books/"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, m.bookViewWithProgress(m.readingAccount(r), lib, book))
}

func (m *multiServer) apiBookSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || m.cfg.DisableKoSyncWebReaderBridge || (m.kosync == nil && m.kosyncRemote == nil) {
		http.NotFound(w, r)
		return
	}
	remainder := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/books/"), "/sync")
	lib, book, ok := m.findBook(remainder)
	if !ok {
		http.NotFound(w, r)
		return
	}
	document, ready := "", false
	if m.publications != nil {
		document, ready = publicationDigest(m.publications, lib.ID, book)
	}
	if !ready {
		acquisition, err := lib.Store.Acquire(r.Context(), book)
		if readerAt, ok := acquisition.Reader.(io.ReaderAt); err == nil && ok {
			document = koSyncPartialMD5(readerAt)
			ready = true
			_ = acquisition.Reader.Close()
		} else if acquisition.Reader != nil {
			_ = acquisition.Reader.Close()
		}
	}
	if !ready {
		writeJSON(w, http.StatusAccepted, map[string]any{"ready": false})
		return
	}
	var progress koSyncProgress
	var exists bool
	matchedBy := "binary"
	var readingPosition readingPosition
	if m.reading != nil {
		readingPosition, _ = m.reading.Get(m.readingAccount(r), lib.ID, book.ID)
	}
	filenameDocument := koSyncFilenameMD5(path.Base(book.Path))
	documents := reading.SyncCandidates(readingPosition, document, filenameDocument)
	for _, candidate := range documents {
		if m.kosyncRemote != nil {
			var err error
			progress, exists, err = m.kosyncRemote.progress(r.Context(), candidate)
			if err != nil {
				http.Error(w, "External reading progress is temporarily unavailable", http.StatusBadGateway)
				return
			}
		} else {
			// The progress being read belongs to the signed-in reader, exactly like
			// the candidate lookup below. Falling back to the configured username
			// here handed every account the owner's position.
			// No KoSync user of their own means no progress to show, never the
			// owner's: this used to fall back to the configured credentials.
			if username := m.koSyncUserFor(m.readingAccount(r)); username != "" {
				progress, exists = m.kosync.progress(username, candidate)
			}
		}
		if exists {
			matchedBy = reading.SyncMatchKind(candidate, readingPosition.KoSyncDocument, filenameDocument)
			break
		}
	}
	response := map[string]any{"ready": true, "document": document}
	if exists {
		response["remote"] = progress
		response["matched_by"] = matchedBy
	} else if m.kosync != nil {
		// The offer belongs to the signed-in reader, and only a position in THIS
		// book's own documents may be offered: linking another book's document
		// would bind the wrong hash to this one.
		username := m.koSyncUserFor(m.readingAccount(r))
		if username == "" {
			username = m.cfg.KoSyncUsername
		}
		if username == "" {
			username = m.cfg.Username
		}
		if candidate, ok := m.kosync.latestProgressAmong(username, documents); ok && reading.ShouldOfferCandidate(readingPosition, readingProgress(candidate), time.Now(), 15*time.Minute) {
			response["candidate"] = candidate
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (m *multiServer) bookView(lib Library, book catalog.Book) bookView {
	key := url.PathEscape(lib.ID) + "/" + url.PathEscape(book.ID)
	coverURL := "/covers/" + key
	if modified := coverModified(book); !modified.IsZero() {
		coverURL += "?v=" + strconv.FormatInt(modified.Unix(), 36)
	}
	pageStreamURL := ""
	if lib.PageStreaming && book.PageCount > 0 && book.PageMediaType != "" {
		if _, ok := catalog.PageCapability(lib.Store); ok {
			pageStreamURL = "/stream/" + key + "?page={pageNumber}"
		}
	}
	return bookView{Book: book, LibraryID: lib.ID, LibraryName: lib.Name, LibraryKind: lib.Kind, CoverURL: coverURL, ReadURL: "/reader/" + key, DownloadURL: "/books/" + key + "/" + url.PathEscape(path.Base(book.Path)), PageStreamURL: pageStreamURL, Bucket: catalog.Bucket(book.Title)}
}

func (m *multiServer) bookViewWithProgress(account string, lib Library, book catalog.Book) bookView {
	view := m.bookView(lib, book)
	if m.reading == nil {
		return view
	}
	position, exists := m.reading.Get(account, lib.ID, book.ID)
	if exists {
		view.Reading = &bookReadingProgressView{Fraction: position.Fraction, UpdatedAt: position.UpdatedAt, Finished: position.Finished, Status: position.Status}
	}
	return view
}

func coverModified(book catalog.Book) time.Time {
	if !book.CoverModified.IsZero() {
		return book.CoverModified
	}
	return book.Modified
}

func (m *multiServer) library(id string) (Library, bool) {
	for _, library := range m.libraries {
		if library.ID == id {
			return library, true
		}
	}
	return Library{}, false
}

func (m *multiServer) findBook(remainder string) (Library, catalog.Book, bool) {
	parts := strings.SplitN(strings.Trim(remainder, "/"), "/", 2)
	if len(parts) != 2 {
		return Library{}, catalog.Book{}, false
	}
	libID, e1 := url.PathUnescape(parts[0])
	bookID, e2 := url.PathUnescape(parts[1])
	if e1 != nil || e2 != nil {
		return Library{}, catalog.Book{}, false
	}
	lib, ok := m.library(libID)
	if ok && lib.Store != nil {
		book, found := lib.Store.Book(bookID)
		return lib, book, found
	}
	return Library{}, catalog.Book{}, false
}
