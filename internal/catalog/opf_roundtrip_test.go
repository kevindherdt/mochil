package catalog

import (
	"encoding/xml"
	"testing"

	"mochil/internal/opf"
)

// What the sidecar writer produces must read back, through the catalogue's own reader, as the same book.
func TestAWrittenSidecarReadsBackAsTheSameBook(t *testing.T) {
	index := 2.5
	raw, err := opf.Generate(opf.Metadata{
		Title: `Dune & "Messiah" <1>`, TitleSort: "Dune Messiah", Authors: []string{"Frank Herbert", "Brian Herbert"}, AuthorSort: "Herbert, Frank & Herbert, Brian",
		Series: "Dune", SeriesIndex: &index, Tags: []string{"Sci-Fi", "Classic"}, Publisher: "Ace", Published: "1969-10-01T00:00:00+00:00", Language: "eng", Rating: 9,
		Description: "Spice & sand.", Identifiers: map[string]string{"isbn": "9780441172719", "hardcover": "dune-messiah"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var pkg packageXML
	if err := xml.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	book := withExtraMetadata(Book{Title: clean(first(pkg.Metadata.Titles)), Language: clean(first(pkg.Metadata.Languages)), Description: clean(first(pkg.Metadata.Descriptions))}, pkg)
	book.Authors, book.AuthorSort = epubCreators(pkg)
	book.Series, book.SeriesIndex = epubSeries(pkg)
	if book.Title != `Dune & "Messiah" <1>` || len(book.Authors) != 2 || book.AuthorSort[0] != "Herbert, Frank" || book.Series != "Dune" || book.SeriesIndex == nil || *book.SeriesIndex != 2.5 {
		t.Fatalf("%+v", book)
	}
	if len(book.Tags) != 2 || book.Publisher != "Ace" || book.Published != "1969-10-01T00:00:00+00:00" || book.Rating != 9 || book.TitleSort != "Dune Messiah" || book.Language != "eng" || book.Description != "Spice & sand." {
		t.Fatalf("%+v", book)
	}
	if book.Identifiers["isbn"] != "9780441172719" || book.Identifiers["hardcover"] != "dune-messiah" || len(book.Identifiers) != 2 {
		t.Fatalf("%v", book.Identifiers)
	}
}
