package catalog

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestOverrideStoreRejectsTrailingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overrides.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"libraries":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenOverrideStore(path); err == nil {
		t.Fatal("override state with trailing JSON was accepted")
	}
}

type overrideTestBackend struct{ book Book }

func (b *overrideTestBackend) Scan() (Stats, error) { return Stats{Books: 1}, nil }
func (b *overrideTestBackend) Books() []Book        { return []Book{b.book} }
func (b *overrideTestBackend) Book(id string) (Book, bool) {
	return b.book, id == b.book.ID
}
func (b *overrideTestBackend) Acquire(context.Context, Book) (Acquisition, error) {
	return Acquisition{}, nil
}

func TestOverrideStorePersistsAndDecoratesBackend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata-overrides.json")
	store, err := OpenOverrideStore(path)
	if err != nil {
		t.Fatal(err)
	}
	backend := store.Wrap("books", &overrideTestBackend{book: Book{ID: "one", Title: "Original", Authors: []string{"Unknown"}}})
	if err := store.Apply("books", map[string]map[string]string{"one": {"authors": "Writer & Illustrator", "series_index": "2.5"}}); err != nil {
		t.Fatal(err)
	}
	book, ok := backend.Book("one")
	if !ok || len(book.Authors) != 2 || book.Authors[0] != "Writer" || book.SeriesIndex == nil || *book.SeriesIndex != 2.5 {
		t.Fatalf("decorated book = %#v", book)
	}
	reopened, err := OpenOverrideStore(path)
	if err != nil {
		t.Fatal(err)
	}
	book, _ = reopened.Wrap("books", backend.Underlying()).Book("one")
	if len(book.Authors) != 2 || book.SeriesIndex == nil || *book.SeriesIndex != 2.5 {
		t.Fatalf("persisted override = %#v", book)
	}
}

func TestOverrideStoreRejectsInvalidBatchAtomically(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "metadata-overrides.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = store.Apply("books", map[string]map[string]string{
		"one": {"authors": "Writer"},
		"two": {"series_index": "not-a-number"},
	})
	if err == nil {
		t.Fatal("invalid batch was accepted")
	}
	book := store.apply("books", Book{ID: "one", Authors: []string{"Unknown"}})
	if book.Authors[0] != "Unknown" {
		t.Fatalf("partial override survived: %#v", book)
	}
}

func TestOverrideStorePersistsReversibleArtwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata-overrides.json")
	store, err := OpenOverrideStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetArtwork("books", "one", "https://assets.hardcover.app/cover.jpg", "Hardcover", "42"); err != nil {
		t.Fatal(err)
	}
	value, ok := store.Get("books", "one")
	if !ok || value.ArtworkURL == "" || value.Provider != "Hardcover" || value.ProviderID != "42" {
		t.Fatalf("artwork override = %#v, %v", value, ok)
	}
	reopened, err := OpenOverrideStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok = reopened.Get("books", "one"); !ok || value.ArtworkURL == "" {
		t.Fatalf("persisted artwork override = %#v, %v", value, ok)
	}
	if err := reopened.DeleteArtwork("books", "one"); err != nil {
		t.Fatal(err)
	}
	if value, _ = reopened.Get("books", "one"); value.ArtworkURL != "" || value.Provider != "" || value.ProviderID != "" {
		t.Fatalf("artwork override was not removed: %#v", value)
	}
}

func TestOverrideCanSetASeriesAndResetItBackToTheFiles(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "o.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := store.Wrap("books", &overrideTestBackend{book: Book{ID: "one", Title: "Original", Series: "Old", Authors: []string{"A"}}})
	if err := store.Apply("books", map[string]map[string]string{"one": {"series": "New Series", "title": "Edited", "series_index": "3"}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := backend.Book("one"); got.Series != "New Series" || got.Title != "Edited" || *got.SeriesIndex != 3 {
		t.Fatalf("%+v", got)
	}
	undo, err := store.Reset("books", "one", "series")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := backend.Book("one"); got.Series != "Old" || got.Title != "Edited" {
		t.Fatalf("only the series should be reset: %+v", got)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if got, _ := backend.Book("one"); got.Series != "New Series" {
		t.Fatalf("undo did not restore the edit: %+v", got)
	}
	if _, err := store.Reset("books", "one"); err != nil {
		t.Fatal(err)
	}
	if got, _ := backend.Book("one"); got.Title != "Original" || got.Series != "Old" || got.SeriesIndex != nil {
		t.Fatalf("a full reset left edits behind: %+v", got)
	}
	if _, err := store.Reset("books", "one", "colour"); err == nil {
		t.Fatal("an unknown field was accepted")
	}
	reopened, _ := OpenOverrideStore(store.path)
	if _, ok := reopened.Get("books", "one"); ok {
		t.Fatal("a reset book still has a stored override")
	}
}

func TestOverridesCarryTheCalibreStyleFields(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "o.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := store.Wrap("books", &overrideTestBackend{book: Book{ID: "one", Title: "T", Tags: []string{"old"}, Rating: 2}})
	if err := store.Apply("books", map[string]map[string]string{"one": {
		"tags": "Fantasy, Light Novel", "publisher": "Yen Press", "published": "2021-05-04", "language": "en", "rating": "4.5",
		"description": "A story.", "identifiers": "ISBN:9781975327071, hardcover:dune", "title_sort": "T, The", "author_sort": "Herbert, Frank & Other, A",
	}}); err != nil {
		t.Fatal(err)
	}
	got, _ := backend.Book("one")
	if len(got.Tags) != 2 || got.Tags[0] != "Fantasy" || got.Publisher != "Yen Press" || got.Published != "2021-05-04" || got.Language != "en" || got.Rating != 9 ||
		got.Description != "A story." || got.Identifiers["isbn"] != "9781975327071" || got.Identifiers["hardcover"] != "dune" || got.TitleSort != "T, The" || len(got.AuthorSort) != 2 {
		t.Fatalf("%+v", got)
	}
	reopened, _ := OpenOverrideStore(store.path)
	if value, ok := reopened.Get("books", "one"); !ok || value.Rating == nil || *value.Rating != 9 {
		t.Fatalf("not persisted: %+v", value)
	}
	for name, change := range map[string]map[string]string{"rating too high": {"rating": "6"}, "rating text": {"rating": "great"}, "identifier without scheme": {"identifiers": "9781975327071"}, "empty identifiers": {"identifiers": " , "}} {
		if err := store.Apply("books", map[string]map[string]string{"one": change}); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := store.Reset("books", "one", "tags", "rating", "identifiers"); err != nil {
		t.Fatal(err)
	}
	got, _ = backend.Book("one")
	if len(got.Tags) != 1 || got.Tags[0] != "old" || got.Rating != 2 || got.Identifiers != nil || got.Publisher != "Yen Press" {
		t.Fatalf("reset should restore only the named fields: %+v", got)
	}
	if _, err := store.Reset("books", "one"); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Get("books", "one"); ok {
		t.Fatal("a full reset left something behind")
	}
}

// snapshotTestBackend publishes one catalogue generation and shares it the way
// Store does, so a reader can tell a handed-out slice from a copy.
type snapshotTestBackend struct {
	mu      sync.Mutex
	books   []Book
	version uint64
}

func (b *snapshotTestBackend) Scan() (Stats, error) { return Stats{Books: len(b.books)}, nil }
func (b *snapshotTestBackend) Books() []Book {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneBooks(b.books)
}
func (b *snapshotTestBackend) Snapshot() []Book {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.books
}
func (b *snapshotTestBackend) CatalogVersion() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.version
}
func (b *snapshotTestBackend) Acquire(context.Context, Book) (Acquisition, error) {
	return Acquisition{}, nil
}
func (b *snapshotTestBackend) Book(id string) (Book, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, book := range b.books {
		if book.ID == id {
			return book, true
		}
	}
	return Book{}, false
}
func (b *snapshotTestBackend) publish(books ...Book) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.books = books
	b.version++
}

// Laying overrides over 5 430 books is ~2 MB of copying that only matters when
// there is an override to lay. A library nobody has corrected must hand out the
// published catalogue itself.
func TestOverrideSnapshotDoesNotCopyALibraryWithoutOverrides(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "o.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := &snapshotTestBackend{books: []Book{{ID: "one", Title: "One", Authors: []string{"A"}}}, version: 1}
	wrapped := store.Wrap("books", backend)
	books := wrapped.Snapshot()
	if len(books) != 1 || &books[0] != &backend.books[0] {
		t.Fatalf("a library with no overrides was copied: %#v", books)
	}
}

// The snapshot is remembered for a generation, so it has to notice both things
// that start a new one: the catalogue moving on and an override being written.
func TestOverrideSnapshotFollowsAScanAndAnEdit(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "o.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := &snapshotTestBackend{books: []Book{{ID: "one", Title: "Original", Authors: []string{"A"}}}, version: 1}
	wrapped := store.Wrap("books", backend)

	if books := wrapped.Snapshot(); len(books) != 1 || books[0].Title != "Original" {
		t.Fatalf("first read = %#v", books)
	}
	if err := store.Apply("books", map[string]map[string]string{"one": {"title": "Edited"}}); err != nil {
		t.Fatal(err)
	}
	if books := wrapped.Snapshot(); len(books) != 1 || books[0].Title != "Edited" {
		t.Fatalf("an edit was served from the previous generation: %#v", books)
	}
	backend.publish(Book{ID: "one", Title: "Rescanned", Authors: []string{"A"}}, Book{ID: "two", Title: "Second", Authors: []string{"B"}})
	books := wrapped.Snapshot()
	// The new generation is there, and the override still sits over its book.
	if len(books) != 2 || books[0].Title != "Edited" || books[1].Title != "Second" {
		t.Fatalf("a scan was served from the previous generation: %#v", books)
	}
	if books[0].Path != "" || books[1].ID != "two" {
		t.Fatalf("the rescanned records did not reach the snapshot: %#v", books)
	}
	// An override written against the new generation lands on its own book.
	undo, err := store.ApplyWithUndo("books", map[string]map[string]string{"two": {"title": "Edited Too"}})
	if err != nil {
		t.Fatal(err)
	}
	if books := wrapped.Snapshot(); len(books) != 2 || books[1].Title != "Edited Too" || books[0].Title != "Edited" {
		t.Fatalf("a new generation lost or misplaced an override: %#v", books)
	}
	if err := undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
}

// A backend that cannot say which generation it published is never remembered:
// there is no way to notice a rescan, and serving a stale catalogue is worse
// than copying it.
func TestOverrideSnapshotIsNotRememberedForAnUnversionedCatalogue(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "o.json"))
	if err != nil {
		t.Fatal(err)
	}
	backend := &overrideTestBackend{book: Book{ID: "one", Title: "Original", Authors: []string{"A"}}}
	wrapped := store.Wrap("books", backend)
	if books := wrapped.Snapshot(); len(books) != 1 || books[0].Title != "Original" {
		t.Fatalf("first read = %#v", books)
	}
	backend.book.Title = "Rescanned"
	if books := wrapped.Snapshot(); len(books) != 1 || books[0].Title != "Rescanned" {
		t.Fatalf("unversioned catalogue served from a cache: %#v", books)
	}
}

func TestOverrideSnapshotAnswersConcurrentReadersWhileTheGenerationMoves(t *testing.T) {
	store, err := OpenOverrideStore(filepath.Join(t.TempDir(), "o.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Apply("books", map[string]map[string]string{"one": {"title": "Edited"}}); err != nil {
		t.Fatal(err)
	}
	backend := &snapshotTestBackend{books: []Book{{ID: "one", Title: "Original", Authors: []string{"A"}}}, version: 1}
	wrapped := store.Wrap("books", backend)
	var wait sync.WaitGroup
	for reader := 0; reader < 8; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for pass := 0; pass < 200; pass++ {
				books := wrapped.Snapshot()
				if len(books) == 0 || books[0].Title != "Edited" {
					t.Errorf("read %d books under a moving generation: %#v", len(books), books)
					return
				}
			}
		}()
	}
	for scan := 0; scan < 50; scan++ {
		backend.publish(Book{ID: "one", Title: "Original", Authors: []string{"A"}})
	}
	wait.Wait()
}
