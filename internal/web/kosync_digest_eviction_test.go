package web

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"mochil/internal/catalog"
)

// removableBackend is a catalogue whose books can disappear between digest
// refreshes, which is what a deleted book looks like to the digest cache.
type removableBackend struct {
	mu    sync.Mutex
	books []catalog.Book
}

func (b *removableBackend) Scan() (catalog.Stats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return catalog.Stats{Books: len(b.books)}, nil
}

func (b *removableBackend) Books() []catalog.Book {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]catalog.Book(nil), b.books...)
}

func (b *removableBackend) Book(id string) (catalog.Book, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, book := range b.books {
		if book.ID == id {
			return book, true
		}
	}
	return catalog.Book{}, false
}

func (b *removableBackend) Acquire(_ context.Context, book catalog.Book) (catalog.Acquisition, error) {
	return catalog.Acquisition{Reader: staticPublication{bytes.NewReader([]byte(book.Title))}}, nil
}

func (b *removableBackend) remove(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	kept := b.books[:0]
	for _, book := range b.books {
		if book.ID != id {
			kept = append(kept, book)
		}
	}
	b.books = kept
}

func TestKoSyncDigestsForgotARemovedBook(t *testing.T) {
	backend := &removableBackend{books: []catalog.Book{
		{ID: "a", Path: "Alpha.epub", Title: "Alpha", Size: 5},
		{ID: "b", Path: "Beta.epub", Title: "Beta", Size: 4},
	}}
	server := &multiServer{libraries: []Library{{ID: "books", Name: "Books", Kind: "books", Store: backend}}}
	server.refreshKoSyncDigests()

	alpha := koSyncPartialMD5(bytes.NewReader([]byte("Alpha")))
	if matches := server.koSyncDigestMatches(alpha); len(matches) != 1 {
		t.Fatalf("matches = %v, want the indexed book", matches)
	}

	backend.remove("a")
	server.refreshKoSyncDigests()

	if matches := server.koSyncDigestMatches(alpha); len(matches) != 0 {
		t.Fatalf("matches = %v, want none: a deleted book must not keep matching its content hash", matches)
	}
	beta := koSyncPartialMD5(bytes.NewReader([]byte("Beta")))
	if matches := server.koSyncDigestMatches(beta); len(matches) != 1 {
		t.Fatalf("matches = %v, want the book that is still there", matches)
	}
}

// An interrupted pass must not be read as "every other book was deleted".
func TestKoSyncDigestsKeepEntriesWhenAnIndexedBookCannotBeRead(t *testing.T) {
	backend := &removableBackend{books: []catalog.Book{
		{ID: "a", Path: "Alpha.epub", Title: "Alpha", Size: 5},
		{ID: "b", Path: "Beta.epub", Title: "Beta", Size: 4},
	}}
	server := &multiServer{libraries: []Library{
		{ID: "books", Name: "Books", Kind: "books", Store: backend},
		{ID: "gone", Name: "Gone", Kind: "books", Store: nil},
	}}
	server.refreshKoSyncDigests()
	backend.remove("a")
	server.refreshKoSyncDigests()

	alpha := koSyncPartialMD5(bytes.NewReader([]byte("Alpha")))
	if matches := server.koSyncDigestMatches(alpha); len(matches) != 1 {
		t.Fatalf("matches = %v, want the entry kept: an unreachable library is not a deletion", matches)
	}
}
