package catalog

import (
	"strings"
)

// PreferredBackend filters equivalent cross-format publications from the
// published snapshot while retaining the underlying records for direct access.
// It never collapses two files of the same format.
type PreferredBackend struct {
	Backend
	formats map[string]int
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

func (p *PreferredBackend) Books() []Book {
	books := p.Backend.Books()
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

func (p *PreferredBackend) CatalogVersion() uint64 {
	if backend, ok := p.Backend.(VersionedBackend); ok {
		return backend.CatalogVersion()
	}
	return 0
}

func (p *PreferredBackend) MetadataHydrating() bool {
	backend, ok := p.Backend.(MetadataHydrationBackend)
	return ok && backend.MetadataHydrating()
}
