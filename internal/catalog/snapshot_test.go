package catalog

import "testing"

func TestCatalogueSnapshotsDoNotExposeNestedBookStorage(t *testing.T) {
	index := 2.0
	original := Book{ID: "one", Authors: []string{"Author"}, AuthorSort: []string{"Author, A"}, SeriesIndex: &index}
	stores := []struct {
		name string
		new  func() Backend
	}{
		{name: "filesystem", new: func() Backend { return &Store{books: []Book{original}, byID: map[string]Book{"one": original}} }},
	}
	for _, test := range stores {
		t.Run(test.name, func(t *testing.T) {
			store := test.new()
			books := store.Books()
			book, ok := store.Book("one")
			if !ok {
				t.Fatal("book not found")
			}
			books[0].Authors[0] = "Mutated"
			books[0].AuthorSort[0] = "Mutated"
			*book.SeriesIndex = 99

			freshBooks := store.Books()
			freshBook, _ := store.Book("one")
			if freshBooks[0].Authors[0] != "Author" || freshBooks[0].AuthorSort[0] != "Author, A" || *freshBook.SeriesIndex != 2 {
				t.Fatalf("stored book was mutated through a snapshot: %+v / %+v", freshBooks[0], freshBook)
			}
		})
	}
}

// A Calibre sidecar edit changes only tags, publisher, rating or identifiers: the
// book file keeps its size and mtime, so an equality test that skips those fields
// made the rescan a no-op and the new metadata never reached the catalogue.
func TestBookEqualSeesEveryPublishedField(t *testing.T) {
	base := Book{ID: "one", Path: "one.epub", Title: "One", Authors: []string{"A"}, Tags: []string{"alpha"}, Publisher: "P", Rating: 6, TitleSort: "One, The", Identifiers: map[string]string{"isbn": "9780000000001"}}
	equal := base
	if !bookEqual(base, equal) {
		t.Fatal("identical books compared unequal")
	}
	changes := map[string]func(*Book){
		"tags":        func(b *Book) { b.Tags = []string{"beta"} },
		"publisher":   func(b *Book) { b.Publisher = "Q" },
		"published":   func(b *Book) { b.Published = "1999" },
		"rating":      func(b *Book) { b.Rating = 8 },
		"identifiers": func(b *Book) { b.Identifiers = map[string]string{"isbn": "9780000000002"} },
		"title sort":  func(b *Book) { b.TitleSort = "One" },
		"read along":  func(b *Book) { b.ReadAlong = true },
	}
	for name, change := range changes {
		other := base
		change(&other)
		if bookEqual(base, other) {
			t.Errorf("a %s-only change compared equal, so the scan would discard it", name)
		}
	}
}
