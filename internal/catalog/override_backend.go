package catalog

import (
	"context"
	"sync"
	"time"
)

// This file is the overlaid view of a library: the store above persists the
// corrections, and OverrideBackend presents a catalogue with them applied.

type OverrideBackend struct {
	libraryID string
	backend   Backend
	overrides *OverrideStore

	// published remembers the overlaid catalogue of one generation. Laying the
	// overrides over 5 430 books costs ~2 MB and 1.5 ms, and every feed and
	// every editor chunk asks for this snapshot; a catalogue changes only when
	// a scan or an override does. The slice is shared read-only, the same
	// contract Store.Snapshot documents, and is dropped when the generation
	// moves on. CatalogVersion carries the override store's generation, so an
	// edit invalidates it as well.
	mu        sync.Mutex
	published []Book
	version   uint64
	cached    bool
}

func (s *OverrideStore) Wrap(libraryID string, backend Backend) *OverrideBackend {
	return &OverrideBackend{libraryID: libraryID, backend: backend, overrides: s}
}

func (b *OverrideBackend) Underlying() Backend  { return b.backend }
func (b *OverrideBackend) Scan() (Stats, error) { return b.backend.Scan() }
func (b *OverrideBackend) Acquire(ctx context.Context, book Book) (Acquisition, error) {
	return b.backend.Acquire(ctx, book)
}
func (b *OverrideBackend) Books() []Book {
	books := b.backend.Books()
	for index := range books {
		books[index] = b.overrides.apply(b.libraryID, books[index])
	}
	return books
}

// Snapshot is Books without the deep clone: when the library has overrides the
// wrapped catalogue is copied shallowly so they can be laid over it, and the
// published books it shares backing storage with are never written to. A
// library with no overrides is returned as published, without the copy.
func (b *OverrideBackend) Snapshot() []Book {
	version, versioned := b.catalogVersion()
	if versioned {
		b.mu.Lock()
		if b.cached && b.version == version {
			published := b.published
			b.mu.Unlock()
			return published
		}
		b.mu.Unlock()
	}
	source := publishedSnapshot(b.backend)
	books := source
	if b.overrides.HasOverrides(b.libraryID) {
		books = make([]Book, len(source))
		copy(books, source)
		for index := range books {
			books[index] = b.overrides.apply(b.libraryID, books[index])
		}
	}
	if versioned {
		b.mu.Lock()
		// Only remember the generation this result actually describes, the way
		// PreferredBackend does: a scan or an override landing mid-call must not
		// be cached under the older generation.
		if current, ok := b.catalogVersion(); ok && current == version {
			b.published, b.version, b.cached = books, version, true
		}
		b.mu.Unlock()
	}
	return books
}

// publishedSnapshot reads the catalogue without copying it, for a caller that
// only lists or counts books.
func publishedSnapshot(backend Backend) []Book {
	if published, ok := backend.(interface{ Snapshot() []Book }); ok {
		return published.Snapshot()
	}
	return backend.Books()
}

// Summary passes the count and newest modification through: an override changes
// a book's metadata, never whether it exists or when its file changed.
func (b *OverrideBackend) Summary() (int, time.Time) {
	if summarized, ok := b.backend.(interface {
		Summary() (int, time.Time)
	}); ok {
		return summarized.Summary()
	}
	books := b.backend.Books()
	return len(books), latestModified(books)
}
func (b *OverrideBackend) Book(id string) (Book, bool) {
	book, ok := b.backend.Book(id)
	if ok {
		book = b.overrides.apply(b.libraryID, book)
	}
	return book, ok
}
func (b *OverrideBackend) CatalogVersion() uint64 {
	version, _ := b.catalogVersion()
	return version
}

// catalogVersion reports one number for the overlaid catalogue: the wrapped
// backend's own generation plus the override store's. An unversioned backend
// reports only the overrides, and is never cached.
func (b *OverrideBackend) catalogVersion() (uint64, bool) {
	var version uint64
	versioned := false
	if backend, ok := b.backend.(VersionedBackend); ok {
		version, versioned = backend.CatalogVersion(), true
	}
	b.overrides.mu.RLock()
	generation := b.overrides.generation
	b.overrides.mu.RUnlock()
	return version + generation, versioned
}

func unwrapBackend(backend Backend) Backend {
	for {
		wrapper, ok := backend.(interface{ Underlying() Backend })
		if !ok {
			return backend
		}
		backend = wrapper.Underlying()
	}
}

func CoverCapability(backend Backend) (CoverBackend, bool) {
	provider, ok := unwrapBackend(backend).(CoverBackend)
	return provider, ok
}

func PageCapability(backend Backend) (PageBackend, bool) {
	provider, ok := unwrapBackend(backend).(PageBackend)
	return provider, ok
}

func MetadataHydrationCapability(backend Backend) (MetadataHydrationBackend, bool) {
	provider, ok := unwrapBackend(backend).(MetadataHydrationBackend)
	return provider, ok
}
