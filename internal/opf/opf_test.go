package opf

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func index(value float64) *float64 { return &value }

var sample = Metadata{
	Title: `Dune & "Messiah" <1>`, TitleSort: "Dune Messiah", Authors: []string{"Frank Herbert", "Brian Herbert"}, AuthorSort: "Herbert, Frank & Herbert, Brian",
	Series: "Dune", SeriesIndex: index(2), Tags: []string{"Sci-Fi", "Classic"}, Publisher: "Ace", Published: "1969-10-01T00:00:00+00:00", Language: "eng", Rating: 9,
	Description: "Spice & sand.", Identifiers: map[string]string{"isbn": "9780441172719", "hardcover": "dune-messiah"},
}

func TestGeneratedSidecarIsWellFormedAndKeepsEveryField(t *testing.T) {
	raw, err := Generate(sample)
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Metadata struct {
			Titles   []string `xml:"title"`
			Creators []struct {
				Name   string `xml:",chardata"`
				FileAs string `xml:"file-as,attr"`
			} `xml:"creator"`
			Subjects    []string `xml:"subject"`
			Publisher   string   `xml:"publisher"`
			Date        string   `xml:"date"`
			Description string   `xml:"description"`
			Language    string   `xml:"language"`
			Identifiers []struct {
				Scheme string `xml:"scheme,attr"`
				Value  string `xml:",chardata"`
			} `xml:"identifier"`
			Metas []struct {
				Name    string `xml:"name,attr"`
				Content string `xml:"content,attr"`
			} `xml:"meta"`
		} `xml:"metadata"`
	}
	if err := xml.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("not well-formed: %v\n%s", err, raw)
	}
	m := pkg.Metadata
	if m.Titles[0] != sample.Title || len(m.Creators) != 2 || m.Creators[1].FileAs != "Herbert, Brian" || len(m.Subjects) != 2 || m.Publisher != "Ace" || m.Description != "Spice & sand." || m.Language != "eng" {
		t.Fatalf("%+v", m)
	}
	ids := map[string]string{}
	for _, id := range m.Identifiers {
		ids[id.Scheme] = id.Value
	}
	if ids["isbn"] != "9780441172719" || ids["hardcover"] != "dune-messiah" || ids["uuid"] == "" || len(ids) != 3 {
		t.Fatalf("%v", ids)
	}
	if !strings.Contains(string(raw), `name="calibre:series" content="Dune"`) || !strings.Contains(string(raw), `name="calibre:series_index" content="2"`) || !strings.Contains(string(raw), `name="calibre:rating" content="9.0"`) || !strings.Contains(string(raw), `calibre:title_sort" content="Dune Messiah"`) {
		t.Fatalf("calibre meta missing:\n%s", raw)
	}
	again, _ := Generate(sample)
	if string(again) != string(raw) {
		t.Fatal("the same fields gave different bytes")
	}
}

func TestSidecarRefusesAnEmptyTitleAndOmitsWhatIsEmpty(t *testing.T) {
	if _, err := Generate(Metadata{Title: "  "}); err == nil {
		t.Fatal("a sidecar without a title was written")
	}
	raw, _ := Generate(Metadata{Title: "Plain", Authors: []string{"A"}, SeriesIndex: index(3)})
	for _, absent := range []string{"calibre:series", "calibre:rating", "dc:publisher", "dc:subject", "dc:description"} {
		if strings.Contains(string(raw), absent) {
			t.Errorf("%s was written for an empty field", absent)
		}
	}
}

func TestAnIdentifierCannotBreakOutOfItsAttribute(t *testing.T) {
	raw, _ := Generate(Metadata{Title: "T", Identifiers: map[string]string{`x"><evil/><y z="`: "1", "uuid": "ignored"}})
	var any struct{}
	if err := xml.Unmarshal(raw, &any); err != nil || strings.Contains(string(raw), "<evil/>") {
		t.Fatalf("%v\n%s", err, raw)
	}
}

func TestWriteBesideNeverReplacesAndPicksTheRightName(t *testing.T) {
	dir := t.TempDir()
	alone := filepath.Join(dir, "One", "One.epub")
	shared := filepath.Join(dir, "Series", "Series - Vol 1.epub")
	os.MkdirAll(filepath.Dir(alone), 0o755)
	os.MkdirAll(filepath.Dir(shared), 0o755)
	os.WriteFile(alone, []byte("x"), 0o644)
	os.WriteFile(shared, []byte("x"), 0o644)
	os.WriteFile(filepath.Join(filepath.Dir(shared), "Series - Vol 2.epub"), []byte("y"), 0o644)
	got, err := WriteBeside(alone, sample, false)
	if err != nil || filepath.Base(got) != "metadata.opf" {
		t.Fatalf("alone: %q %v", got, err)
	}
	got, err = WriteBeside(shared, sample, false)
	if err != nil || filepath.Base(got) != "Series - Vol 1.opf" {
		t.Fatalf("shared folder should get its own file: %q %v", got, err)
	}
	original, _ := os.ReadFile(got)
	changed := sample
	changed.Title = "Changed"
	if _, err := WriteBeside(shared, changed, false); !errors.Is(err, ErrExists) {
		t.Fatalf("an existing sidecar was not protected: %v", err)
	}
	if now, _ := os.ReadFile(got); string(now) != string(original) {
		t.Fatal("the existing sidecar changed")
	}
	if _, err := WriteBeside(shared, changed, true); err != nil {
		t.Fatal(err)
	}
	if now, _ := os.ReadFile(got); !strings.Contains(string(now), "Changed") {
		t.Fatal("an explicit overwrite did not write")
	}
	if leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(shared), ".opf-*")); len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}
	if _, err := WriteBeside(filepath.Join(dir, "missing.epub"), sample, false); err == nil {
		t.Fatal("a sidecar was written for a book that does not exist")
	}
}
