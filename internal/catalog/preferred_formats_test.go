package catalog

import (
	"context"
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
