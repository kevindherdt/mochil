package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mochil/internal/jobs"
	"mochil/internal/statefile"
)

type Book struct {
	ID              string            `json:"id"`
	Path            string            `json:"path"`
	Title           string            `json:"title"`
	Authors         []string          `json:"authors"`
	AuthorSort      []string          `json:"author_sort,omitempty"`
	Series          string            `json:"series,omitempty"`
	SeriesIndex     *float64          `json:"series_index,omitempty"`
	PublicationRole PublicationRole   `json:"publication_role,omitempty"`
	Format          string            `json:"format,omitempty"`
	MediaType       string            `json:"media_type,omitempty"`
	PageCount       int               `json:"page_count,omitempty"`
	PageMediaType   string            `json:"page_media_type,omitempty"`
	Language        string            `json:"language,omitempty"`
	Description     string            `json:"description,omitempty"`
	Identifier      string            `json:"identifier,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	Publisher       string            `json:"publisher,omitempty"`
	Published       string            `json:"published,omitempty"`
	Rating          int               `json:"rating,omitempty"` // 0 to 10, as Calibre stores it (two per star)
	Identifiers     map[string]string `json:"identifiers,omitempty"`
	TitleSort       string            `json:"title_sort,omitempty"`
	Size            int64             `json:"size"`
	Modified        time.Time         `json:"modified"`
	CoverModified   time.Time         `json:"cover_modified,omitempty"`
	MetadataRead    bool              `json:"metadata_read,omitempty"`
	ReadAlong       bool              `json:"read_along,omitempty"` // an EPUB with its own recorded narration
}

type cacheFile struct {
	Version     int              `json:"version"`
	Fingerprint string           `json:"fingerprint,omitempty"`
	Revision    string           `json:"revision,omitempty"`
	Books       []Book           `json:"books"`
	Sidecars    map[string]int64 `json:"sidecars,omitempty"`
	Scanned     time.Time        `json:"scanned"`
}

const catalogCacheVersion = 9

type Stats struct {
	Books, Read, Reused, Errors int
	FirstError                  string
}

func (s *Stats) recordError(operation, path string, err error) {
	s.Errors++
	if s.FirstError == "" {
		s.FirstError = fmt.Sprintf("%s %q: %v", operation, path, err)
	}
}

type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type Acquisition struct {
	Reader      ReadSeekCloser
	RedirectURL string
}

type Backend interface {
	Scan() (Stats, error)
	Books() []Book
	Book(id string) (Book, bool)
	Acquire(context.Context, Book) (Acquisition, error)
}

// VersionedBackend exposes a monotonically increasing catalogue generation.
// Query layers can use it to rebuild derived indexes only when a backend has
// published a new immutable book snapshot.
type VersionedBackend interface {
	CatalogVersion() uint64
}

// MetadataHydrationBackend reports whether a remote catalogue is still
// enriching lightweight directory entries with embedded publication metadata.
type MetadataHydrationBackend interface {
	MetadataHydrating() bool
}

type Page struct {
	Reader    io.ReadCloser
	MediaType string
	Size      int64
}

type PageBackend interface {
	Page(context.Context, Book, int) (Page, error)
}

// CoverBackend lets remote stores fetch an archive cover without downloading
// the complete publication. Backends may omit it and use acquisition fallback.
type CoverBackend interface {
	Cover(context.Context, Book) ([]byte, error)
}

type Store struct {
	library string
	cache   string
	options FilesystemOptions
	mu      sync.RWMutex
	scanMu  sync.Mutex
	books   []Book
	byID    map[string]Book
	version uint64
	// count and latest describe the published snapshot, maintained when it is
	// replaced so a landing page can ask without copying every book.
	count  int
	latest time.Time
	// pagesMu guards the sorted page order of recently streamed archives, keyed
	// by path, size and modification time.
	pagesMu   sync.Mutex
	pageLists map[pageCacheKey][]int
	pageQueue []pageCacheKey
}

func (s *Store) CatalogVersion() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

type FilesystemOptions struct {
	FolderLayout   string
	FilenameAuthor bool
	Calibre        bool
}

func NewStore(library, cache string) *Store {
	return NewFilesystemStore(library, cache, FilesystemOptions{FolderLayout: "auto"})
}

func NewFilesystemStore(library, cache string, options FilesystemOptions) *Store {
	if options.FolderLayout == "" {
		options.FolderLayout = "auto"
	}
	return &Store{library: library, cache: cache, options: options}
}

func (s *Store) Books() []Book {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneBooks(s.books)
}

// Snapshot returns the published books without copying them. A landing page
// that only lists titles or counts entries does not need Books' deep copy.
//
// It is safe because replaceWithIndex publishes a whole new slice at once and
// never writes into the slice it replaced: a caller holding this one sees a
// consistent generation, and the next publish leaves it untouched. Callers
// must treat it as read-only -- sort or edit a copy, such as the one Books
// returns, when the result has to change.
func (s *Store) Snapshot() []Book {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.books
}

// Summary reports the published book count and the newest modification time
// without touching the books themselves. Computing those from Books cost a
// clone of the whole catalogue on every OPDS landing page.
func (s *Store) Summary() (int, time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count, s.latest
}

func latestModified(books []Book) time.Time {
	var latest time.Time
	for _, book := range books {
		if book.Modified.After(latest) {
			latest = book.Modified
		}
	}
	return latest
}

func (s *Store) Book(id string) (Book, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.byID[id]
	return cloneBook(b), ok
}

func (s *Store) AbsolutePath(b Book) string {
	return filepath.Join(s.library, filepath.FromSlash(b.Path))
}

func (s *Store) safeAbsolutePath(b Book) (string, error) {
	root, err := filepath.Abs(s.library)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(b.Path)))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("indexed path escapes the library")
	}
	info, err := os.Lstat(candidate)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("indexed path became a symbolic link")
	}
	return candidate, nil
}

func (s *Store) Acquire(_ context.Context, b Book) (Acquisition, error) {
	path, err := s.safeAbsolutePath(b)
	if err != nil {
		return Acquisition{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Acquisition{}, err
	}
	return Acquisition{Reader: f}, nil
}

func (s *Store) Scan() (Stats, error) {
	return s.ScanContext(context.Background())
}

func (s *Store) ScanContext(ctx context.Context) (Stats, error) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	root, err := filepath.Abs(s.library)
	if err != nil {
		return Stats{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		cached := s.loadCache()
		if len(cached.Books) > 0 && cached.Fingerprint == s.fingerprint() {
			s.replace(cached.Books)
			return Stats{Books: len(cached.Books), Reused: len(cached.Books), Errors: 1}, nil
		}
		return Stats{}, fmt.Errorf("library: %w", err)
	}
	if !info.IsDir() {
		return Stats{}, fmt.Errorf("library is not a directory: %s", root)
	}

	old := s.loadCache()
	if old.Fingerprint != s.fingerprint() {
		old = cacheFile{}
	}
	known := make(map[string]Book, len(old.Books))
	for _, b := range old.Books {
		known[b.Path] = b
	}
	var books []Book
	var stats Stats
	sidecars := make(map[string]int64)
	calibreChoices := make(map[string]string)
	var candidates []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			stats.recordError("walk", path, walkErr)
			return nil
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		extension := strings.ToLower(filepath.Ext(d.Name()))
		supported := extension == ".epub" || extension == ".cbz" || (s.options.Calibre && (extension == ".pdf" || extension == ".mobi" || extension == ".azw3"))
		if d.IsDir() || d.Type()&os.ModeSymlink != 0 || !supported {
			return nil
		}
		if s.options.Calibre {
			directory := filepath.Dir(path)
			choice, checked := calibreChoices[directory]
			if !checked {
				entries, readErr := os.ReadDir(directory)
				if readErr != nil {
					stats.recordError("read directory", directory, readErr)
					return nil
				}
				for _, entry := range entries {
					if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
						continue
					}
					if choice == "" || calibreFormatRank(entry.Name()) < calibreFormatRank(choice) || (calibreFormatRank(entry.Name()) == calibreFormatRank(choice) && entry.Name() < choice) {
						if calibreFormatRank(entry.Name()) < 100 {
							choice = entry.Name()
						}
					}
				}
				calibreChoices[directory] = choice
			}
			if d.Name() != choice {
				return nil
			}
		}
		candidates = append(candidates, path)
		return nil
	})
	if err != nil {
		return stats, err
	}
	total := int64(len(candidates))
	for index, path := range candidates {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		info, infoErr := os.Stat(path)
		if infoErr != nil {
			stats.recordError("inspect", path, infoErr)
			continue
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			stats.recordError("make relative path", path, relErr)
			continue
		}
		rel = filepath.ToSlash(rel)
		jobs.Report(ctx, int64(index+1), total, "Checking "+rel)
		sidecarStamp := int64(0)
		if s.options.Calibre {
			for _, sidecarName := range []string{"metadata.opf", "cover.jpg"} {
				if sidecarInfo, sidecarErr := os.Stat(filepath.Join(filepath.Dir(path), sidecarName)); sidecarErr == nil {
					sidecarStamp ^= sidecarInfo.ModTime().UnixNano()
				}
			}
			sidecars[rel] = sidecarStamp
		}
		if cached, ok := known[rel]; ok && cached.Size == info.Size() && cached.Modified.Equal(info.ModTime()) && (!s.options.Calibre || old.Sidecars[rel] == sidecarStamp) {
			books = append(books, cached)
			stats.Reused++
			continue
		}
		var book Book
		var readErr error
		if s.options.Calibre {
			book, readErr = readCalibrePublication(path, rel, info)
		} else {
			book, readErr = readPublication(path, rel, info)
		}
		if readErr != nil {
			stats.recordError("read publication", path, readErr)
			continue
		}
		books = append(books, book)
		stats.Read++
	}
	inferFilesystemMetadata(books, s.options)
	sort.Slice(books, func(i, j int) bool { return NaturalLess(books[i].Title, books[j].Title) })
	byID := make(map[string]Book, len(books))
	for _, b := range books {
		byID[b.ID] = b
	}
	s.replaceWithIndex(books, byID)
	stats.Books = len(books)
	jobs.Report(ctx, int64(stats.Books), int64(stats.Books), fmt.Sprintf("Indexed %d files", stats.Books))
	if err := s.saveCache(cacheFile{Version: catalogCacheVersion, Fingerprint: s.fingerprint(), Books: books, Sidecars: sidecars, Scanned: time.Now().UTC()}); err != nil {
		return stats, err
	}
	return stats, nil
}

func (s *Store) replace(books []Book) {
	byID := make(map[string]Book, len(books))
	for _, book := range books {
		byID[book.ID] = book
	}
	s.replaceWithIndex(books, byID)
}

func (s *Store) replaceWithIndex(books []Book, byID map[string]Book) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if booksEqual(s.books, books) {
		return
	}
	s.books, s.byID = books, byID
	s.count, s.latest = len(books), latestModified(books)
	s.version++
}

func (s *Store) fingerprint() string {
	return s.options.FolderLayout + ":" + strconv.FormatBool(s.options.FilenameAuthor) + ":" + strconv.FormatBool(s.options.Calibre)
}

func (s *Store) loadCache() cacheFile {
	var c cacheFile
	if statefile.DecodeJSON(s.cache, statefile.MaximumCatalogueBytes, &c) != nil || c.Version != catalogCacheVersion || len(c.Books) > maxManifestItems {
		return cacheFile{}
	}
	return c
}

func (s *Store) saveCache(c cacheFile) error {
	dir := filepath.Dir(s.cache)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".mochil-cache-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = statefile.EncodeJSON(tmp, statefile.MaximumCatalogueBytes, c, true); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, s.cache); err != nil {
		_ = os.Remove(s.cache)
		err = os.Rename(name, s.cache)
	}
	return err
}

func readPublication(path, rel string, info os.FileInfo) (Book, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub":
		return readEPUB(path, rel, info)
	case ".cbz":
		return readCBZ(path, rel, info)
	default:
		return Book{}, errors.New("unsupported publication format")
	}
}

func bookID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:8])
}
func SortKey(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	for _, article := range []string{"the ", "an ", "a "} {
		if strings.HasPrefix(v, article) {
			return strings.TrimSpace(v[len(article):])
		}
	}
	return v
}

// NaturalLess compares normalized display values while treating digit runs as
// numbers, so "Volume 2" sorts before "Volume 10".
func NaturalLess(left, right string) bool {
	return NaturalKeyLess(SortKey(left), SortKey(right))
}

// NaturalKeyLess compares values that have already passed through SortKey.
// Derived indexes use it inside hot sort loops without repeatedly allocating
// normalized copies of the same titles.
func NaturalKeyLess(a, b string) bool {
	for len(a) > 0 && len(b) > 0 {
		ad, bd := a[0] >= '0' && a[0] <= '9', b[0] >= '0' && b[0] <= '9'
		if ad && bd {
			ai, bi := 0, 0
			for ai < len(a) && a[ai] >= '0' && a[ai] <= '9' {
				ai++
			}
			for bi < len(b) && b[bi] >= '0' && b[bi] <= '9' {
				bi++
			}
			an, bn := strings.TrimLeft(a[:ai], "0"), strings.TrimLeft(b[:bi], "0")
			if an == "" {
				an = "0"
			}
			if bn == "" {
				bn = "0"
			}
			if len(an) != len(bn) {
				return len(an) < len(bn)
			}
			if an != bn {
				return an < bn
			}
			a, b = a[ai:], b[bi:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func Bucket(value string) string {
	k := strings.ToUpper(SortKey(value))
	if k == "" {
		return "#"
	}
	r := []rune(k)[0]
	if r >= 'A' && r <= 'Z' {
		return string(r)
	}
	return "#"
}
