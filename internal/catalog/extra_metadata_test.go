package catalog

import (
	"encoding/xml"
	"testing"
)

func TestExtraMetadataIsReadLikeCalibreKeepsIt(t *testing.T) {
	var pkg packageXML
	raw := `<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
<dc:title>T</dc:title><dc:subject>Fantasy</dc:subject><dc:subject> Light Novel </dc:subject><dc:publisher>Yen Press</dc:publisher><dc:date>2021-05-04T00:00:00+00:00</dc:date>
<dc:identifier opf:scheme="calibre">1234</dc:identifier><dc:identifier opf:scheme="uuid">u-1</dc:identifier><dc:identifier opf:scheme="ISBN">9781975327071</dc:identifier><dc:identifier opf:scheme="hardcover">dune</dc:identifier>
<meta name="calibre:rating" content="8.0"/><meta name="calibre:title_sort" content="Title, The"/></metadata></package>`
	if err := xml.Unmarshal([]byte(raw), &pkg); err != nil {
		t.Fatal(err)
	}
	book := withExtraMetadata(Book{}, pkg)
	if len(book.Tags) != 2 || book.Tags[1] != "Light Novel" || book.Publisher != "Yen Press" || book.Published != "2021-05-04T00:00:00+00:00" || book.Rating != 8 || book.TitleSort != "Title, The" {
		t.Fatalf("%+v", book)
	}
	if book.Identifiers["isbn"] != "9781975327071" || book.Identifiers["hardcover"] != "dune" || len(book.Identifiers) != 2 {
		t.Fatalf("calibre's own and uuid ids should be left out: %v", book.Identifiers)
	}
	if firstIdentifier(pkg) != "1234" {
		t.Fatalf("the first identifier stays available: %q", firstIdentifier(pkg))
	}
	if bad := withExtraMetadata(Book{}, packageXML{}); bad.Tags != nil || bad.Identifiers != nil {
		t.Fatalf("an empty package should add nothing: %+v", bad)
	}
}
