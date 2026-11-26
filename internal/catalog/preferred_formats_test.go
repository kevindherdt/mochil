package catalog

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

type preferredTestBackend struct{ books []Book }

func (b *preferredTestBackend) Scan() (Stats, error) { return Stats{Books: len(b.books)}, nil }
func (b *preferredTestBackend) Books() []Book        { return append([]Book(nil), b.books...) }
func (b *preferredTestBackend) Acquire(context.Context, Book) (Acquisition, error) {
	return Acquisition{}, nil
}
func (b *preferredTestBackend) Book(id string) (Book, bool) {
	for _, book := range b.books {
		if book.ID == id {
			return book, true
		}
	}
	return Book{}, false
}

func TestPreferredBackendKeepsHighestRankedEquivalentFormat(t *testing.T) {
	index := 1.0
	backend := &preferredTestBackend{books: []Book{
		{ID: "pdf", Title: "Book One", Authors: []string{"Writer"}, Series: "Series", SeriesIndex: &index, Format: "pdf"},
		{ID: "epub", Title: "Book One", Authors: []string{"Writer"}, Series: "Series", SeriesIndex: &index, Format: "epub"},
		{ID: "other", Title: "Different Book", Authors: []string{"Writer"}, Format: "pdf"},
	}}
	preferred := PreferFormats(backend, []string{"epub", "pdf"})
	books := preferred.Books()
	if len(books) != 2 || books[0].ID != "epub" || books[1].ID != "other" {
		t.Fatalf("preferred books = %#v", books)
	}
	if alternate, ok := preferred.Book("pdf"); !ok || alternate.ID != "pdf" {
		t.Fatal("alternate format is no longer directly addressable")
	}
}

func TestPreferredBackendDoesNotCollapseWeakOrSameFormatMatches(t *testing.T) {
	backend := &preferredTestBackend{books: []Book{
		{ID: "unknown-pdf", Title: "Unattributed", Authors: []string{"Unknown"}, Format: "pdf"},
		{ID: "unknown-epub", Title: "Unattributed", Authors: []string{"Unknown"}, Format: "epub"},
		{ID: "first", Title: "Edition", Authors: []string{"Writer"}, Format: "epub"},
		{ID: "second", Title: "Edition", Authors: []string{"Writer"}, Format: "epub"},
	}}
	if books := PreferFormats(backend, []string{"epub", "pdf"}).Books(); len(books) != 4 {
		t.Fatalf("weak or same-format matches were collapsed: %#v", books)
	}
}

func TestPreferredBackendKeepsConflictingLanguagesAndIdentifiers(t *testing.T) {
	backend := &preferredTestBackend{books: []Book{
		{ID: "english", Title: "The Book", Authors: []string{"Writer"}, Format: "epub", Language: "en", Identifier: "urn:isbn:1111111111"},
		{ID: "french", Title: "The Book", Authors: []string{"Writer"}, Format: "pdf", Language: "fr", Identifier: "2222222222"},
		{ID: "ambiguous", Title: "The Book", Authors: []string{"Writer"}, Format: "mobi"},
	}}
	books := PreferFormats(backend, []string{"epub", "pdf", "mobi"}).Books()
	if len(books) != 3 {
		t.Fatalf("distinct or ambiguous editions collapsed: %#v", books)
	}
}

func TestPreferredBackendSeparatesEachEditionDiscriminator(t *testing.T) {
	for name, books := range map[string][]Book{
		"language": {
			{ID: "one", Title: "Shared", Authors: []string{"Writer"}, Format: "epub", Language: "en", Identifier: "same"},
			{ID: "two", Title: "Shared", Authors: []string{"Writer"}, Format: "pdf", Language: "nl", Identifier: "same"},
		},
		"identifier": {
			{ID: "one", Title: "Shared", Authors: []string{"Writer"}, Format: "epub", Language: "en", Identifier: "first-edition"},
			{ID: "two", Title: "Shared", Authors: []string{"Writer"}, Format: "pdf", Language: "en", Identifier: "revised-edition"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := PreferFormats(&preferredTestBackend{books: books}, []string{"epub", "pdf"}).Books(); len(got) != 2 {
				t.Fatalf("edition discriminator collapsed: %#v", got)
			}
		})
	}
}

func TestPreferredBackendAllowsOneKnownEditionToMatchMissingMetadata(t *testing.T) {
	backend := &preferredTestBackend{books: []Book{
		{ID: "epub", Title: "The Book", Authors: []string{"Writer"}, Format: "epub", Language: "en", Identifier: "urn:isbn:123-456-789-X"},
		{ID: "pdf", Title: "The Book", Authors: []string{"Writer"}, Format: "pdf"},
	}}
	books := PreferFormats(backend, []string{"epub", "pdf"}).Books()
	if len(books) != 1 || books[0].ID != "epub" {
		t.Fatalf("ordinary incomplete cross-format match failed: %#v", books)
	}
}

func TestPreferredBackendKeepsDifferentPublicationRoles(t *testing.T) {
	backend := &preferredTestBackend{books: []Book{
		{ID: "main", Title: "Saga", Authors: []string{"Writer"}, Format: "epub"},
		{ID: "omnibus", Title: "Saga", Authors: []string{"Writer"}, Format: "pdf", PublicationRole: PublicationOmnibus},
	}}
	if books := PreferFormats(backend, []string{"epub", "pdf"}).Books(); len(books) != 2 {
		t.Fatalf("publication roles collapsed: %#v", books)
	}
}

// versionedTestBackend publishes a catalogue generation the way Store does, and
// counts how often the preferred filter had to look at it. Both counters are
// atomic because Store reads its version through a lock while readers run.
type versionedTestBackend struct {
	preferredTestBackend
	version atomic.Uint64
	reads   atomic.Int64
	mu      sync.Mutex
}

func (b *versionedTestBackend) Books() []Book {
	b.reads.Add(1)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.preferredTestBackend.Books()
}

func (b *versionedTestBackend) CatalogVersion() uint64 { return b.version.Load() }

// publish moves the catalogue to its next generation.
func (b *versionedTestBackend) publish(books []Book) {
	b.mu.Lock()
	b.books = books
	b.mu.Unlock()
	b.version.Add(1)
}

func editionPair(preferredFormat string) []Book {
	index := 1.0
	return []Book{
		{ID: "epub", Title: "Book One", Authors: []string{"Writer"}, Series: "Series", SeriesIndex: &index, Format: "epub"},
		{ID: "pdf", Title: "Book One", Authors: []string{"Writer"}, Series: "Series", SeriesIndex: &index, Format: preferredFormat},
	}
}

func TestPreferredBackendReusesThePublishedGeneration(t *testing.T) {
	backend := &versionedTestBackend{preferredTestBackend: preferredTestBackend{books: editionPair("pdf")}}
	preferred := PreferFormats(backend, []string{"epub", "pdf"}).(*PreferredBackend)
	first := preferred.Books()
	second := preferred.Snapshot()
	if backend.reads.Load() != 1 {
		t.Fatalf("the same generation was filtered %d times", backend.reads.Load())
	}
	if len(second) != 1 || second[0].ID != "epub" || first[0].ID != second[0].ID {
		t.Fatalf("cached snapshot = %#v, first read = %#v", second, first)
	}
}

func TestPreferredBackendRepartitionsWhenTheGenerationChanges(t *testing.T) {
	backend := &versionedTestBackend{preferredTestBackend: preferredTestBackend{books: editionPair("pdf")}}
	preferred := PreferFormats(backend, []string{"epub", "pdf"}).(*PreferredBackend)
	if books := preferred.Snapshot(); len(books) != 1 || books[0].ID != "epub" {
		t.Fatalf("first generation = %#v", books)
	}
	// A rescan drops the EPUB, so the PDF becomes the only edition: serving
	// the remembered answer here would hide the book entirely.
	backend.publish(editionPair("pdf")[1:])
	books := preferred.Snapshot()
	if len(books) != 1 || books[0].ID != "pdf" {
		t.Fatalf("stale generation served: %#v", books)
	}
}

func TestPreferredBackendDoesNotRememberAnUnversionedCatalogue(t *testing.T) {
	backend := &preferredTestBackend{books: editionPair("pdf")}
	preferred := PreferFormats(backend, []string{"epub", "pdf"}).(*PreferredBackend)
	if books := preferred.Snapshot(); len(books) != 1 || books[0].ID != "epub" {
		t.Fatalf("first read = %#v", books)
	}
	backend.books = editionPair("pdf")[1:]
	if books := preferred.Snapshot(); len(books) != 1 || books[0].ID != "pdf" {
		t.Fatalf("unversioned catalogue served from a cache: %#v", books)
	}
}

func TestPreferredBackendSnapshotIsSharedWhileBooksIsACopy(t *testing.T) {
	backend := &versionedTestBackend{preferredTestBackend: preferredTestBackend{books: []Book{
		{ID: "one", Title: "Book One", Authors: []string{"Writer"}, Tags: []string{"tag"}, Format: "epub"},
	}}}
	preferred := PreferFormats(backend, []string{"epub"}).(*PreferredBackend)
	edited := preferred.Books()
	edited[0].Title = "Renamed"
	edited[0].Authors[0] = "Someone Else"
	edited[0].Tags[0] = "other"
	assertIntact := func(when string) {
		t.Helper()
		books := preferred.Snapshot()
		if len(books) != 1 || books[0].Title != "Book One" || books[0].Authors[0] != "Writer" || books[0].Tags[0] != "tag" {
			t.Fatalf("%s: edited copy reached the catalogue: %#v", when, books)
		}
	}
	assertIntact("after editing Books")
	assertIntact("on the next read")
}

func TestPreferredBackendAnswersConcurrentReadersWhileTheGenerationMoves(t *testing.T) {
	backend := &versionedTestBackend{preferredTestBackend: preferredTestBackend{books: editionPair("pdf")}}
	preferred := PreferFormats(backend, []string{"epub", "pdf"}).(*PreferredBackend)
	var wait sync.WaitGroup
	for reader := 0; reader < 8; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for pass := 0; pass < 200; pass++ {
				if books := preferred.Books(); len(books) == 0 {
					t.Errorf("empty generation")
					return
				}
				_ = preferred.Snapshot()
			}
		}()
	}
	for scan := 0; scan < 50; scan++ {
		backend.publish(editionPair("pdf"))
	}
	wait.Wait()
}
