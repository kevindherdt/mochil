package catalog

import (
	"strings"
	"testing"
)

// The `lightnovels25` drop made every 86 title read "-EIGHTY-SIX - Volume 01". The series folder was right
// ("86--EIGHTY-SIX"), the position was right (1), and only the title was mangled — by this parser, not by the AI and
// not by anything the person did.
//
// `bareNumberedFilename` (`^(\d+)\s*[-_]\s*(.+)$`) saw the leading `86` of the series' own name, took it as a volume
// number, and handed back the rest of the name as the title: `86--EIGHTY-SIX - Volume 01` → 86 + `-EIGHTY-SIX - Volume
// 01`. The double dash is the giveaway: a rule that produces a title beginning with a separator has mis-parsed, and
// should refuse rather than invent.
func TestALeadingNumberThatBelongsToTheTitleIsNotAVolumeNumber(t *testing.T) {
	for _, name := range []string{
		"86--EIGHTY-SIX - Volume 01",
		"86--EIGHTY-SIX - Side Stories - Volume 02",
		"1984 - A Novel",          // "1984" is the title, not volume 1984
		"11/22/63 - A Novel",      // a number that is the whole title
		"-EIGHTY-SIX - Volume 01", // already mangled once: do not mangle it further
	} {
		if parsed := parseNumberedFilename(name); parsed.ok {
			t.Errorf("%q parsed as volume %v, title %q — a bare leading number is not a volume unless what is left "+
				"after it reads as a title", name, parsed.index, parsed.title)
		}
	}
}

// The rule still has to work for the shape it was written for: a number that really is a position, followed by the
// book's own name.
func TestABareNumberFollowedByARealTitleStillParses(t *testing.T) {
	for _, test := range []struct {
		name  string
		index float64
		title string
	}{
		{"01 Dies the Fire", 1, "01 Dies the Fire"}, // no separator, so not this rule's business
		{"1 - The Fellowship of the Ring", 1, "The Fellowship of the Ring"},
		{"12_The Long Way", 12, "The Long Way"},
		{"3.5 - Interlude", 3.5, "Interlude"},
	} {
		parsed := parseNumberedFilename(test.name)
		if test.name == "01 Dies the Fire" {
			// The marked/bracket rules do not match and the bare rule needs a separator; either way nothing is taken
			// from a name that is just a zero-padded position and a title with a space.
			if parsed.ok && parsed.index != 1 {
				t.Errorf("%q parsed as volume %v, want 1", test.name, parsed.index)
			}
			continue
		}
		if !parsed.ok || parsed.index != test.index || parsed.title != test.title {
			t.Errorf("%q parsed as %+v, want volume %v title %q", test.name, parsed, test.index, test.title)
		}
	}
}

// The whole scan, on the real folder shape: the series and the position come out right, and the title no longer starts
// with the dash that started this.
func TestScanningTheEightySixFolderGivesAReadableTitleAndTheRightPosition(t *testing.T) {
	books := []Book{
		{Path: "lightnovels25/86--EIGHTY-SIX/86--EIGHTY-SIX - Volume 07.epub", Title: "86--EIGHTY-SIX - Volume 07"},
		{Path: "lightnovels25/86--EIGHTY-SIX/86--EIGHTY-SIX - Volume 08.epub", Title: "86--EIGHTY-SIX - Volume 08"},
		{Path: "lightnovels25/86--EIGHTY-SIX/86--EIGHTY-SIX - Side Stories - Volume 01.epub", Title: "86--EIGHTY-SIX - Side Stories - Volume 01"},
	}
	inferFilesystemMetadata(books, FilesystemOptions{})

	for _, book := range books {
		if strings.HasPrefix(book.Title, "-") {
			t.Errorf("%q kept the leading dash the parser used to leave behind", book.Title)
		}
		if book.Series != "86--EIGHTY-SIX" {
			t.Errorf("series = %q, want the folder's own name", book.Series)
		}
	}
	if books[0].SeriesIndex == nil || *books[0].SeriesIndex != 7 {
		t.Errorf("volume = %v, want 7 from the marked form in the name", books[0].SeriesIndex)
	}
	// A side story is classified as such and deliberately left without a position: it belongs to the series but is not
	// volume 1 of it. Folding these into a sub-series ("86--EIGHTY-SIX — Side Stories") is a decision for the cleanup
	// pass, where it can be previewed per row — not something a scan should guess while nobody is looking.
	if books[2].PublicationRole != "side_story" {
		t.Errorf("side stories role = %q, want it classified as a side story", books[2].PublicationRole)
	}
	if books[2].SeriesIndex != nil {
		t.Errorf("side stories volume = %v, want none", *books[2].SeriesIndex)
	}
}
