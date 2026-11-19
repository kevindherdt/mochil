package web

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"mochil/internal/catalog"
)

type Config struct {
	Listen, Library, Cache, PublicURL, Username, Password string
	BasePath, AcquisitionPath                             string
	StreamingPath                                         string
	PageStreaming                                         bool
	LibraryName                                           string
}

type Server struct {
	cfg   Config
	store catalog.Backend
	log   *slog.Logger
}

func New(cfg Config, store catalog.Backend, logger *slog.Logger) (http.Handler, error) {
	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, errors.New("username and password must be configured together")
	}
	if cfg.PublicURL != "" {
		u, err := url.Parse(cfg.PublicURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, errors.New("public-url must be an absolute HTTP(S) URL")
		}
		cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "/opds"
	}
	if cfg.AcquisitionPath == "" {
		cfg.AcquisitionPath = "/books"
	}
	if cfg.StreamingPath == "" {
		cfg.StreamingPath = "/stream"
	}
	s := &Server{cfg: cfg, store: store, log: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.root)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /opensearch.xml", s.openSearch)
	mux.HandleFunc("GET /opds", s.catalog)
	mux.HandleFunc("GET /opds/series", s.series)
	mux.HandleFunc("GET /opds/authors", s.authors)
	mux.HandleFunc("GET /opds/titles", s.titles)
	mux.HandleFunc("GET /opds/recent", s.recent)
	mux.HandleFunc("GET /opds/search", s.search)
	mux.HandleFunc("GET /books/{id}/{name}", s.download)
	mux.HandleFunc("GET /stream/{id}", s.streamPage)
	var handler http.Handler = mux
	if cfg.Username != "" {
		handler = s.basicAuth(handler)
	}
	return handler, nil
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, s.catalogHref(""), http.StatusTemporaryRedirect)
}
func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	books := publishedBooks(s.store)
	title := s.cfg.LibraryName
	if title == "" {
		title = "Library"
	}
	feed := s.feed(title, "urn:opdshelf:root", books)
	feed.Links = append(feed.Links, Link{Rel: "search", Href: s.catalogHref("/opensearch.xml"), Type: "application/opensearchdescription+xml"})
	items := []struct{ title, path, description string }{
		{"Series", "/series", "Browse series by initial letter"},
		{"Authors", "/authors", "Browse authors by initial letter"},
		{"Titles", "/titles", "Browse titles by initial letter"},
		{"Recently Added", "/recent", "Newest files first"},
	}
	for _, item := range items {
		feed.Entries = append(feed.Entries, navEntry(item.title, item.description, s.catalogHref(item.path), updated(books), item.path))
	}
	s.writeFeed(w, feed, "navigation")
}

func (s *Server) series(w http.ResponseWriter, r *http.Request) {
	books := publishedBooks(s.store)
	bucket := strings.ToUpper(r.URL.Query().Get("bucket"))
	name := r.URL.Query().Get("name")
	if bucket == "" {
		s.bucketFeed(w, "Series", "/series", books, func(b catalog.Book) string { return b.Series })
		return
	}
	if name == "" {
		counts := map[string]int{}
		for _, b := range books {
			if b.Series != "" && catalog.Bucket(b.Series) == bucket {
				counts[b.Series]++
			}
		}
		names := sortedCountKeys(counts)
		feed := s.feed("Series - "+bucket, "urn:opdshelf:series:"+bucket, books)
		when := updated(books)
		for _, series := range names {
			href := s.queryHref("/series", url.Values{"bucket": {bucket}, "name": {series}})
			feed.Entries = append(feed.Entries, navEntry(series, fmt.Sprintf("%d books", counts[series]), href, when, "series:"+series))
		}
		s.writeFeed(w, feed, "navigation")
		return
	}
	var matches []catalog.Book
	for _, b := range books {
		if b.Series == name {
			matches = append(matches, b)
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return seriesLess(matches[i], matches[j]) })
	s.writeBooks(w, name, "urn:opdshelf:series:"+name, matches)
}

func (s *Server) authors(w http.ResponseWriter, r *http.Request) {
	books := publishedBooks(s.store)
	bucket := strings.ToUpper(r.URL.Query().Get("bucket"))
	name := r.URL.Query().Get("name")
	if bucket == "" {
		s.bucketFeed(w, "Authors", "/authors", books, func(b catalog.Book) string {
			if len(b.AuthorSort) > 0 {
				return b.AuthorSort[0]
			}
			return ""
		})
		return
	}
	if name == "" {
		counts := map[string]int{}
		for _, b := range books {
			for _, a := range b.AuthorSort {
				if catalog.Bucket(a) == bucket {
					counts[a]++
				}
			}
		}
		feed := s.feed("Authors - "+bucket, "urn:opdshelf:authors:"+bucket, books)
		when := updated(books)
		for _, author := range sortedCountKeys(counts) {
			href := s.queryHref("/authors", url.Values{"bucket": {bucket}, "name": {author}})
			feed.Entries = append(feed.Entries, navEntry(author, fmt.Sprintf("%d books", counts[author]), href, when, "author:"+author))
		}
		s.writeFeed(w, feed, "navigation")
		return
	}
	var matches []catalog.Book
	for _, b := range books {
		if contains(b.AuthorSort, name) {
			matches = append(matches, b)
		}
	}
	s.writeBooks(w, name, "urn:opdshelf:author:"+name, matches)
}

func (s *Server) titles(w http.ResponseWriter, r *http.Request) {
	books := publishedBooks(s.store)
	bucket := strings.ToUpper(r.URL.Query().Get("bucket"))
	if bucket == "" {
		s.bucketFeed(w, "Titles", "/titles", books, func(b catalog.Book) string { return b.Title })
		return
	}
	var matches []catalog.Book
	for _, b := range books {
		if catalog.Bucket(b.Title) == bucket {
			matches = append(matches, b)
		}
	}
	s.writeBooks(w, "Titles - "+bucket, "urn:opdshelf:titles:"+bucket, matches)
}

func (s *Server) recent(w http.ResponseWriter, _ *http.Request) {
	// publishedBooks is read-only and shared with the store: sort a copy, or
	// this feed would reorder the catalogue under every other request.
	books := append([]catalog.Book(nil), publishedBooks(s.store)...)
	sort.Slice(books, func(i, j int) bool { return books[i].Modified.After(books[j].Modified) })
	if len(books) > 100 {
		books = books[:100]
	}
	s.writeBooks(w, "Recently Added", "urn:opdshelf:recent", books)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	books := publishedBooks(s.store)
	matches := make([]catalog.Book, 0)
	if q != "" {
		for _, b := range books {
			if strings.Contains(strings.ToLower(b.Title+" "+strings.Join(b.Authors, " ")+" "+strings.Join(b.AuthorSort, " ")+" "+b.Series), q) {
				matches = append(matches, b)
			}
		}
	}
	s.writeBooks(w, "Search: "+q, "urn:opdshelf:search:"+q, matches)
}

func (s *Server) bucketFeed(w http.ResponseWriter, title, path string, books []catalog.Book, value func(catalog.Book) string) {
	counts := map[string]int{}
	for _, b := range books {
		if value(b) != "" {
			counts[catalog.Bucket(value(b))]++
		}
	}
	feed := s.feed(title, "urn:opdshelf:"+strings.ToLower(title), books)
	for _, bucket := range append([]string{"#"}, alphabet()...) {
		if counts[bucket] == 0 {
			continue
		}
		href := s.queryHref(path, url.Values{"bucket": {bucket}})
		feed.Entries = append(feed.Entries, navEntry(bucket, fmt.Sprintf("%d books", counts[bucket]), href, updated(books), path+":"+bucket))
	}
	s.writeFeed(w, feed, "navigation")
}

func (s *Server) writeBooks(w http.ResponseWriter, title, id string, books []catalog.Book) {
	feed := s.feed(title, id, books)
	showIndex := strings.HasPrefix(id, "urn:opdshelf:series:")
	for _, b := range books {
		feed.Entries = append(feed.Entries, s.bookEntry(b, showIndex))
	}
	s.writeFeed(w, feed, "acquisition")
}

func (s *Server) bookEntry(b catalog.Book, showIndex bool) Entry {
	authors := make([]Person, 0, len(b.Authors))
	for _, a := range b.Authors {
		authors = append(authors, Person{Name: a})
	}
	detail := b.Description
	if detail == "" && b.Series != "" {
		detail = b.Series
		if b.SeriesIndex != nil {
			detail += " #" + strconv.FormatFloat(*b.SeriesIndex, 'f', -1, 64)
		}
	}
	name := filepath.Base(filepath.FromSlash(b.Path))
	href := s.href(s.cfg.AcquisitionPath + "/" + b.ID + "/" + url.PathEscape(name))
	title := b.Title
	if showIndex && b.SeriesIndex != nil {
		title = formatSeriesIndex(*b.SeriesIndex) + " - " + title
	}
	mediaType := b.MediaType
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	if b.ReadAlong {
		// An e-reader's list shows only titles, so a book with its own recording says so there.
		title += " (read-along)"
		detail = strings.TrimSpace("Has its own recording. " + detail)
	}
	entry := Entry{ID: "urn:opdshelf:book:" + b.ID, Title: title, Updated: b.Modified.UTC().Format(time.RFC3339), Authors: authors, Content: Content{Type: "text", Text: detail}, Links: []Link{{Rel: acquisitionRel, Href: href, Type: mediaType, Title: "Download"}}}
	if s.cfg.PageStreaming && b.PageCount > 0 && b.PageMediaType != "" {
		if _, ok := catalog.PageCapability(s.store); ok {
			streamHref := s.href(s.cfg.StreamingPath + "/" + b.ID + "?page={pageNumber}")
			entry.Links = append(entry.Links, Link{Rel: "http://vaemendis.net/opds-pse/stream", Href: streamHref, Type: b.PageMediaType, Title: "Stream pages", PSECount: b.PageCount})
		}
	}
	return entry
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	b, ok := s.store.Book(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	acquisition, err := s.store.Acquire(r.Context(), b)
	if err != nil {
		http.Error(w, "Book is temporarily unavailable", http.StatusBadGateway)
		return
	}
	if acquisition.RedirectURL != "" {
		http.Redirect(w, r, acquisition.RedirectURL, http.StatusFound)
		return
	}
	if acquisition.Reader == nil {
		http.Error(w, "Book is temporarily unavailable", http.StatusBadGateway)
		return
	}
	defer acquisition.Reader.Close()
	mediaType := b.MediaType
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filepath.Base(b.Path)))
	http.ServeContent(w, r, filepath.Base(b.Path), b.Modified, acquisition.Reader)
}

func (s *Server) streamPage(w http.ResponseWriter, r *http.Request) {
	backend, ok := catalog.PageCapability(s.store)
	if !s.cfg.PageStreaming || !ok {
		http.NotFound(w, r)
		return
	}
	book, ok := s.store.Book(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	index, err := strconv.Atoi(r.URL.Query().Get("page"))
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

func sortedCountKeys(set map[string]int) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return catalog.SortKey(out[i]) < catalog.SortKey(out[j]) })
	return out
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
func seriesLess(a, b catalog.Book) bool {
	if a.Series == b.Series {
		leftDepth, rightDepth := strings.Count(filepath.ToSlash(a.Path), "/"), strings.Count(filepath.ToSlash(b.Path), "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
	}
	if a.SeriesIndex == nil && b.SeriesIndex != nil {
		return false
	}
	if a.SeriesIndex != nil && b.SeriesIndex == nil {
		return true
	}
	if a.SeriesIndex != nil && b.SeriesIndex != nil && *a.SeriesIndex != *b.SeriesIndex {
		return *a.SeriesIndex < *b.SeriesIndex
	}
	return catalog.NaturalLess(a.Title, b.Title)
}
func formatSeriesIndex(index float64) string {
	if index >= 0 && index < 100 && index == float64(int(index)) {
		return fmt.Sprintf("%02.0f", index)
	}
	return strconv.FormatFloat(index, 'f', -1, 64)
}
func alphabet() []string {
	out := make([]string, 26)
	for i := range out {
		out[i] = string(rune('A' + i))
	}
	return out
}
func (s *Server) href(path string) string {
	if s.cfg.PublicURL != "" {
		return s.cfg.PublicURL + path
	}
	return path
}
func (s *Server) catalogHref(path string) string { return s.href(s.cfg.BasePath + path) }
func (s *Server) queryHref(path string, q url.Values) string {
	return s.catalogHref(path) + "?" + q.Encode()
}
func (s *Server) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		goodUser := subtle.ConstantTimeCompare([]byte(u), []byte(s.cfg.Username)) == 1
		goodPass := subtle.ConstantTimeCompare([]byte(p), []byte(s.cfg.Password)) == 1
		if !ok || !goodUser || !goodPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Mochil", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
