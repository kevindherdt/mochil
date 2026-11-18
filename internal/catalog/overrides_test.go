package catalog

import (
	"context"
	"os"
	"path/filepath"
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
