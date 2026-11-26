package catalog

import (
	"strings"
	"sync"
)

// PreferredBackend filters equivalent cross-format publications from the
// published snapshot while retaining the underlying records for direct access.
// It never collapses two files of the same format.
type PreferredBackend struct {
	Backend
	formats map[string]int

	// published remembers the filtered books of one catalogue generation.
	// Deciding which formats to keep is not cheap: it classifies every book by
	// title and path (publicationPartitions, ~31 ms for 5 400 books), and a
	// catalogue is asked for the same generation many times between scans --
	// every feed, every editor chunk. The published slice is treated as
	// read-only, the way Store.Snapshot is, and is dropped as soon as the
	// generation moves on.
	mu        sync.Mutex
	published []Book
	version   uint64
	cached    bool
}

func PreferFormats(backend Backend, formats []string) Backend {
	if backend == nil || len(formats) == 0 {
		return backend
	}
	ranks := make(map[string]int, len(formats))
	for _, format := range formats {
		format = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(format, ".")))
		if format != "" {
			if _, exists := ranks[format]; !exists {
				ranks[format] = len(ranks)
			}
		}
	}
	if len(ranks) == 0 {
		return backend
	}
	return &PreferredBackend{Backend: backend, formats: ranks}
}

func (p *PreferredBackend) Scan() (Stats, error) {
	stats, err := p.Backend.Scan()
	if err == nil {
		stats.Books = len(p.Books())
	}
	return stats, err
}

// Books returns the preferred books as a deep copy: a caller may edit titles,
// tags or identifiers without reaching anything another reader sees.
func (p *PreferredBackend) Books() []Book {
	return cloneBooks(p.snapshot())
}

// Snapshot returns the preferred books without copying them. They match Books
// and are shared read-only, the same contract Store.Snapshot documents, so a
// feed that only lists or counts books skips both the deep copy and the
// re-partitioning.
func (p *PreferredBackend) Snapshot() []Book {
	return p.snapshot()
}

func (p *PreferredBackend) snapshot() []Book {
	version, versioned := p.catalogVersion()
	p.mu.Lock()
	if versioned && p.cached && p.version == version {
		published := p.published
		p.mu.Unlock()
		return published
	}
	p.mu.Unlock()
	// Books hands back storage this call owns, so the filtered slice can be
	// cached as it is and returned without a second clone.
	books := p.Backend.Books()
	selected := p.selectPreferred(books)
	if versioned {
		p.mu.Lock()
		// Only remember the generation this result actually describes: a scan
		// finishing mid-call would otherwise label newer books with the older
		// version, and a later call for that version would never recompute.
		if current, ok := p.catalogVersion(); ok && current == version {
			p.published, p.version, p.cached = selected, version, true
		}
		p.mu.Unlock()
	}
	return selected
}

func (p *PreferredBackend) selectPreferred(books []Book) []Book {
	partitions := publicationPartitions(books)
	best := make(map[string]int)
	for index, book := range books {
		key := partitions[index]
		if key == "" {
			continue
		}
		rank := p.rank(book.Format)
		if current, exists := best[key]; !exists || rank < current {
			best[key] = rank
		}
	}
	out := make([]Book, 0, len(books))
	for index, book := range books {
		key := partitions[index]
		if key == "" || p.rank(book.Format) == best[key] {
			out = append(out, book)
		}
	}
	return out
}

func (p *PreferredBackend) rank(format string) int {
	if rank, ok := p.formats[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(format, ".")))]; ok {
		return rank
	}
	return len(p.formats) + 1
}

func (p *PreferredBackend) Underlying() Backend { return p.Backend }

func (p *PreferredBackend) catalogVersion() (uint64, bool) {
	backend, ok := p.Backend.(VersionedBackend)
	if !ok {
		return 0, false
	}
	return backend.CatalogVersion(), true
}

func (p *PreferredBackend) CatalogVersion() uint64 {
	version, _ := p.catalogVersion()
	return version
}

func (p *PreferredBackend) MetadataHydrating() bool {
	backend, ok := p.Backend.(MetadataHydrationBackend)
	return ok && backend.MetadataHydrating()
}
