package ingest

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const calibreOPF = `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>Mushoku Tensei, Vol. 3</dc:title><dc:creator>Rifujin na Magonote</dc:creator><dc:language>en</dc:language>
<dc:identifier opf:scheme="ISBN" xmlns:opf="http://www.idpf.org/2007/opf">9781234567890</dc:identifier>
<meta name="calibre:series" content="Mushoku Tensei"/><meta name="calibre:series_index" content="3"/></metadata></package>`

func TestOPFSeriesFromCalibreTags(t *testing.T) {
	got, err := ParseOPF([]byte(calibreOPF))
	if err != nil || got.Title != "Mushoku Tensei, Vol. 3" || got.Series != "Mushoku Tensei" || got.SeriesIndex == nil || *got.SeriesIndex != 3 || got.Authors[0] != "Rifujin na Magonote" || got.Identifiers["isbn"] != "9781234567890" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestOPFSeriesFromEPUB3Collection(t *testing.T) {
	got, _ := ParseOPF([]byte(`<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>B</dc:title>
<meta property="belongs-to-collection" id="c1">Rebuild World</meta><meta refines="#c1" property="group-position">1.2</meta></metadata></package>`))
	if got.Series != "Rebuild World" || got.SeriesIndex == nil || *got.SeriesIndex != 1.2 {
		t.Fatalf("%+v", got)
	}
}

func TestReadEmbeddedOpensARealEPUB(t *testing.T) {
	file := filepath.Join(t.TempDir(), "x.epub")
	out, _ := os.Create(file)
	zw := zip.NewWriter(out)
	w, _ := zw.Create("META-INF/container.xml")
	w.Write([]byte(`<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`))
	w, _ = zw.Create("OEBPS/content.opf")
	w.Write([]byte(calibreOPF))
	zw.Close()
	out.Close()
	got, err := ReadEmbedded(file)
	if err != nil || got.Series != "Mushoku Tensei" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := ReadEmbedded(filepath.Join(t.TempDir(), "missing.epub")); err == nil {
		t.Fatal("a missing file was read")
	}
}

func TestBadNamesAndFoldersStillGetGoodMetadataFromTheBook(t *testing.T) {
	path := InferPath(Publication, "dl/new/ebook (3) final.epub")
	embedded, _ := ParseOPF([]byte(calibreOPF))
	got := Combine(path, embedded)
	if got.Title != "Mushoku Tensei, Vol. 3" || got.Series != "Mushoku Tensei" || *got.SeriesIndex != 3 || got.Authors[0] != "Rifujin na Magonote" || got.Identifiers["isbn"] == "" {
		t.Fatalf("%+v", got)
	}
}

func TestPlaceholdersInTheBookDoNotOverrideTheFolders(t *testing.T) {
	path := InferPath(Publication, "Author Name/Some Series/Some Series - Vol 2 - Title.epub")
	got := Combine(path, Embedded{Title: "Unknown", Authors: []string{"Unknown"}})
	if got.Authors[0] != "Author Name" || got.Series != "Some Series" || got.Title == "Unknown" {
		t.Fatalf("%+v", got)
	}
}

func TestDisagreementBetweenBookAndPathIsFlaggedNotHidden(t *testing.T) {
	path := InferPath(Publication, "Someone Else/Mushoku Tensei/Mushoku Tensei - Vol 4.epub")
	embedded, _ := ParseOPF([]byte(calibreOPF))
	got := Combine(path, embedded)
	if len(got.Conflicts) != 1 || !strings.HasPrefix(got.Conflicts[0], "number:") || *got.SeriesIndex != 3 || got.Authors[0] != "Rifujin na Magonote" {
		t.Fatalf("conflicts=%v index=%v", got.Conflicts, got.SeriesIndex)
	}
	agree := Combine(InferPath(Publication, "Rifujin na Magonote/Mushoku Tensei/Mushoku Tensei - Vol 3.epub"), embedded)
	if len(agree.Conflicts) != 0 {
		t.Fatalf("agreeing sources conflicted: %v", agree.Conflicts)
	}
}

func TestVolumeAndPartBecomeADecimalIndex(t *testing.T) {
	got := InferPath(Publication, "A/Rebuild World/Rebuild World - Volume 1 Part 2.epub")
	if got.SeriesIndex == nil || *got.SeriesIndex != 1.2 {
		t.Fatalf("%v", got.SeriesIndex)
	}
	if got := InferPath(Publication, "A/S/S - Vol 01.2.epub"); got.SeriesIndex == nil || *got.SeriesIndex != 1.2 {
		t.Fatalf("%v", got.SeriesIndex)
	}
}

func TestSideStoriesStayInTheirSeriesAndAreMarkedInTheTitle(t *testing.T) {
	got := InferPath(Publication, "Mushoku Tensei/!Side Stories/Mushoku Tensei - Roxy.epub")
	if got.Series != "Mushoku Tensei" || got.Title != "Roxy (Side Story)" || got.SeriesIndex != nil {
		t.Fatalf("%+v", got)
	}
	again := InferPath(Publication, "Rifujin/Mushoku Tensei/!Side Stories/Roxy Side Story.epub")
	if again.Series != "Mushoku Tensei" || again.Authors[0] != "Rifujin" || again.Title != "Roxy Side Story" {
		t.Fatalf("%+v", again)
	}
}

func TestACategoryFolderIsNeverTakenAsTheSeries(t *testing.T) {
	got := InferPath(Publication, "!Romance/Some Series/Some Series - Vol 2.epub")
	if got.Series != "Some Series" || len(got.Authors) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestABatchFolderIsNotAnAuthorConflictAndAFilenameThatDropsThePrefixIsNotATitleConflict(t *testing.T) {
	path := InferPath(Publication, "lightnovels25/37C/37C - Complete.epub")
	embedded, _ := ParseOPF([]byte(`<package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>37C - Complete</dc:title><dc:creator>Rio Sugihara</dc:creator></metadata></package>`))
	got := Combine(path, embedded)
	if len(got.Conflicts) != 0 || got.Authors[0] != "Rio Sugihara" || got.Series != "37C" {
		t.Fatalf("%+v", got)
	}
	// with nothing in the book, the folder guess is kept but only as low confidence
	bare := Combine(path, Embedded{})
	low := false
	for _, e := range bare.Evidence {
		low = low || (e.Field == "authors" && e.Confidence == Low)
	}
	if !low {
		t.Fatalf("a folder author should be low confidence: %+v", bare.Evidence)
	}
}

func TestPunctuationOnlyTitlesAndYearInSeriesAreNotConflicts(t *testing.T) {
	embedded := Embedded{Title: "Grimgar - Volume 14 - ++", Series: "Naruto Shinden (2018)"}
	path := Proposal{Kind: Publication, Title: "++", Series: "Naruto Shinden"}
	if got := Combine(path, embedded); len(got.Conflicts) != 0 {
		t.Fatalf("%v", got.Conflicts)
	}
	if got := Combine(Proposal{Kind: Publication, Series: "Other Series"}, embedded); len(got.Conflicts) != 1 {
		t.Fatalf("a different series should still conflict: %v", got.Conflicts)
	}
}

func TestAFolderNamedAtTheStartOfTheFilenameIsTheSeries(t *testing.T) {
	got := InferPath(Publication, "Dune/Dune - Volume 02.epub")
	if got.Series != "Dune" || got.SeriesIndex == nil || *got.SeriesIndex != 2 || len(got.Authors) != 0 || got.Title != "" && got.Title == "Dune - Volume 02" {
		t.Fatalf("%+v", got)
	}
	if got := InferPath(Publication, "Random Folder/Something Else.epub"); got.Series != "" {
		t.Fatalf("an unrelated folder became a series: %+v", got)
	}
	if got := InferPath(Publication, "ab/abc.epub"); got.Series != "" {
		t.Fatalf("a very short folder name is not evidence: %+v", got)
	}
}
