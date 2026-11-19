// Package query builds disposable, read-only indexes over catalogue backends.
// It contains catalogue filtering and ordering rules without depending on HTTP,
// OPDS, storage implementation details, or presentation URLs.
package query

import (
	"encoding/binary"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"mochil/internal/catalog"
)

const MaxPageSize = 500

type Library struct {
	ID    string
	Name  string
	Kind  string
	Store catalog.Backend
}

type LibraryResult struct {
	ID    string
	Name  string
	Kind  string
	Count int
}

type BookResult struct {
	LibraryID   string
	LibraryName string
	Book        catalog.Book
}

type BooksRequest struct {
	LibraryID      string
	Query          string
	Series         *string
	Author         *string
	Bucket         string
	Limit          int
	Offset         int
	AfterLibraryID string
	AfterBookID    string
	Sort           string
}

type BooksResult struct {
	Items       []BookResult
	Total       int
	AnchorFound bool
	HasMore     bool
}

type BookNavigationResult struct {
	Bucket string
	Offset int
	Count  int
}

type indexedBook struct {
	book      catalog.Book
	titleKey  string
	bucket    string
	searchKey string
	pathDepth int
}

type libraryIndex struct {
	library Library
	mu      sync.RWMutex
	// buildMu serializes index rebuilds. Without it every request that saw a
	// new catalogue version built the whole index, and all but one were thrown
	// away -- a metadata edit plus a burst of readers cost hundreds of
	// megabytes of short-lived allocations.
	buildMu         sync.Mutex
	initialized     bool
	version         uint64
	books           []indexedBook
	positions       map[string]int
	series          []SeriesResult
	seriesPositions map[string]int
	authors         []AuthorResult
	authorPositions map[string]int
	recent          []int
	recentPositions map[string]int
}

// buildTestHook observes a completed index build. It is nil in production;
// tests use it to prove concurrent readers rebuild once.
var buildTestHook func()

type Service struct {
	libraries []*libraryIndex
}

func New(libraries []Library) *Service {
	service := &Service{libraries: make([]*libraryIndex, 0, len(libraries))}
	for _, library := range libraries {
		service.libraries = append(service.libraries, &libraryIndex{library: library})
	}
	return service
}

func (s *Service) Libraries() []LibraryResult {
	out := make([]LibraryResult, 0, len(s.libraries))
	for _, library := range s.libraries {
		out = append(out, LibraryResult{
			ID: library.library.ID, Name: library.library.Name, Kind: library.library.Kind,
			Count: len(library.snapshot()),
		})
	}
	return out
}

// Version returns a stable generation for the selected publication shelves.
// Cursor callers bind it to their normalized view parameters separately.
func (s *Service) Version(libraryID string) uint64 {
	hash := fnv.New64a()
	var encoded [8]byte
	for _, library := range s.libraries {
		if libraryID != "" && library.library.ID != libraryID {
			continue
		}
		_ = library.snapshot()
		library.mu.RLock()
		version := library.version
		library.mu.RUnlock()
		_, _ = hash.Write([]byte(library.library.ID))
		_, _ = hash.Write([]byte{0})
		binary.LittleEndian.PutUint64(encoded[:], version)
		_, _ = hash.Write(encoded[:])
	}
	version := hash.Sum64()
	if version == 0 {
		return 1
	}
	return version
}

func (i *libraryIndex) snapshot() []indexedBook {
	versioned, hasVersion := i.library.Store.(catalog.VersionedBackend)
	if hasVersion {
		if books, ok := i.published(versioned.CatalogVersion()); ok {
			return books
		}
	}
	i.buildMu.Lock()
	defer i.buildMu.Unlock()
	version := uint64(0)
	if hasVersion {
		// Re-read under the build lock: whoever held it may already have built
		// the version that made this caller wait, and a second build of the
		// same generation would only be discarded.
		version = versioned.CatalogVersion()
		if books, ok := i.published(version); ok {
			return books
		}
	}
	books := i.library.Store.Books()
	indexed := make([]indexedBook, len(books))
	for n, book := range books {
		indexed[n] = indexedBook{book: book, titleKey: catalog.SortKey(book.Title), bucket: catalog.Bucket(book.Title), searchKey: strings.ToLower(book.Title + " " + book.Series + " " + strings.Join(book.Authors, " ")), pathDepth: strings.Count(filepath.ToSlash(book.Path), "/")}
	}
	sort.SliceStable(indexed, func(a, b int) bool {
		if indexed[a].titleKey != indexed[b].titleKey {
			return catalog.NaturalKeyLess(indexed[a].titleKey, indexed[b].titleKey)
		}
		return indexed[a].book.ID < indexed[b].book.ID
	})
	positions := make(map[string]int, len(indexed))
	for position, item := range indexed {
		positions[item.book.ID] = position
	}
	series := buildSeriesIndex(i.library, indexed)
	seriesPositions := make(map[string]int, len(series))
	for position, group := range series {
		seriesPositions[group.Representative.ID] = position
	}
	authors := buildAuthorIndex(i.library, indexed)
	authorPositions := make(map[string]int, len(authors))
	for position, group := range authors {
		authorPositions[group.Name] = position
	}
	recent := make([]int, len(indexed))
	for position := range indexed {
		recent[position] = position
	}
	sort.SliceStable(recent, func(a, b int) bool {
		left, right := indexed[recent[a]].book, indexed[recent[b]].book
		if !left.Modified.Equal(right.Modified) {
			return left.Modified.After(right.Modified)
		}
		return left.ID < right.ID
	})
	recentPositions := make(map[string]int, len(recent))
	for position, index := range recent {
		recentPositions[indexed[index].book.ID] = position
	}
	i.mu.Lock()
	i.books, i.positions, i.series, i.seriesPositions, i.authors, i.authorPositions, i.recent, i.recentPositions, i.version, i.initialized = indexed, positions, series, seriesPositions, authors, authorPositions, recent, recentPositions, version, true
	i.mu.Unlock()
	if buildTestHook != nil {
		buildTestHook()
	}
	return indexed
}

// published returns the index when it already describes this catalogue
// generation.
func (i *libraryIndex) published(version uint64) ([]indexedBook, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.initialized && i.version == version {
		return i.books, true
	}
	return nil, false
}
