package metadata

import (
	"strings"
	"testing"
)

// The rows in these tests are copied out of the real `test` library — the shapes the rules exist for, not invented ones.

func TestATitleLeftWithASeparatorLosesItAndKeepsEverythingElse(t *testing.T) {
	result, changed := Tidy(TidyInput{Title: "-EIGHTY-SIX - Volume 01", Series: "86--EIGHTY-SIX", SeriesIndex: float64Ptr(1)})
	if !changed {
		t.Fatal("a title starting with a dash was left alone")
	}
	// The dash goes — and that is all. This library's series and position are already right, so restructuring the title
	// into "Volume 01" would trade a defect for a title that says less than the one it replaced.
	if result.Title != "EIGHTY-SIX - Volume 01" {
		t.Fatalf("title = %q, want only the dash gone", result.Title)
	}
	if len(result.Rules) != 1 || result.Rules[0] != RuleStripSeparator {
		t.Fatalf("rules = %v, want exactly %s", result.Rules, RuleStripSeparator)
	}
	if result.Unsure {
		t.Fatal("stripping a separator is not a guess")
	}
}

func TestASideStoryBecomesASubSeriesOfItsSeries(t *testing.T) {
	result, changed := Tidy(TidyInput{
		Title:  "-EIGHTY-SIX - Side Stories - Volume 02",
		Series: "86--EIGHTY-SIX",
		Role:   "side_story",
	})
	if !changed {
		t.Fatal("a side story with a leading dash was left alone")
	}
	if result.Series != "86--EIGHTY-SIX — Side Stories" {
		t.Fatalf("series = %q, want the sub-series named under the series", result.Series)
	}
	if result.SeriesIndex == nil || *result.SeriesIndex != 2 {
		t.Fatalf("volume = %v, want 2", result.SeriesIndex)
	}
	if result.Title != "EIGHTY-SIX - Side Stories - Volume 02" {
		t.Fatalf("title = %q, want only the separator gone", result.Title)
	}
}

// When the name after the series carries the book's own title, that title is worth recovering — this is the case where
// restructuring is a real gain rather than a rearrangement.
func TestAPerBookTitleAfterTheVolumeIsRecovered(t *testing.T) {
	// A complete title is left complete. Kevin, twice: the volume has to stay (it is how a book is recognised when the
	// position is wrong) and he does not want the series name taken off either — the title he has is the title he wants,
	// and the only work is removing junk from it.
	result, changed := Tidy(TidyInput{Title: "Baccano! - Volume 04 - 1932 - Drugs & The Dominos", Series: "Baccano!", SeriesIndex: float64Ptr(4)})
	if changed || result.Title != "Baccano! - Volume 04 - 1932 - Drugs & The Dominos" {
		t.Fatalf("changed=%v title=%q, want the title untouched", changed, result.Title)
	}
	// And a volume with no name of its own is left alone, because "Volume 10" is not a title.
	quiet, changed := Tidy(TidyInput{Title: "Ascendance of a Bookworm - Volume 10", Series: "Ascendance of a Bookworm", SeriesIndex: float64Ptr(10)})
	if changed || quiet.Title != "Ascendance of a Bookworm - Volume 10" {
		t.Fatalf("changed=%v title=%q, want the title untouched", changed, quiet.Title)
	}
}

// A filename tag is not a title: stripping the series prefix must not leave "Complete" behind as the book's name.
// The six mistakes the real preview showed, one per shape.
func TestTheSecondIterationOfRealRows(t *testing.T) {
	// A stylised series name keeps its closing dash.
	if got, _ := Tidy(TidyInput{Title: "Re-ZERO -Starting Life in Another World- - Volume 08", Series: "Re-ZERO -Starting Life in Another World-"}); got.Series != "Re-ZERO -Starting Life in Another World-" {
		t.Errorf("series = %q, want the published stylisation", got.Series)
	}
	// A bare number or punctuation after the marker is never the whole title: the volume has to survive, or the row
	// stops saying which book it is.
	for _, title := range []string{"Girls Kingdom - Volume 01 - 02", "Grimgar of Fantasy and Ash - Volume 14 - ++"} {
		got, _ := Tidy(TidyInput{Title: title, Series: strings.SplitN(title, " - ", 2)[0]})
		if !strings.Contains(strings.ToLower(got.Title), "volume") {
			t.Errorf("%q became title %q, want the volume kept", title, got.Title)
		}
	}
	// The series name inside a longer title is not a prefix to remove.
	// It may gain a series position (which is information, not a rewrite); the title itself stays exactly as written.
	title := "Boogiepop - Volume 05 - Boogiepop Overdrive - The King of Distortion"
	if got, _ := Tidy(TidyInput{Title: title, Series: "Boogiepop"}); got.Title != title {
		t.Errorf("title = %q, want it left exactly as it is", got.Title)
	}
	// Initials keep their period.
	if got, _ := Tidy(TidyInput{Title: "Semantic Error", Authors: []string{"Soori J.", "Angy"}}); strings.Join(got.Authors, "/") != "Soori J./Angy" {
		t.Errorf("authors = %v, want the initials untouched", got.Authors)
	}
}

func TestAStatusWordIsNotATitle(t *testing.T) {
	result, changed := Tidy(TidyInput{
		Title:  "As The Villainess, I Reject These Happy-Bad Endings! - Complete",
		Series: "As The Villainess, I Reject These Happy-Bad Endings!",
	})
	if changed || result.Title != "As The Villainess, I Reject These Happy-Bad Endings! - Complete" {
		t.Fatalf("changed=%v title=%q, want the title untouched", changed, result.Title)
	}
}

func TestAVolumeOfANoSuchSeriesNamesTheSeries(t *testing.T) {
	result, _ := Tidy(TidyInput{Title: "Dune - Volume 3"})
	if result.Series != "Dune" || result.SeriesIndex == nil || *result.SeriesIndex != 3 {
		t.Fatalf("series/index = %q/%v, want Dune/3", result.Series, result.SeriesIndex)
	}
}

func TestATitleThatIsJustANumberIsNotAVolume(t *testing.T) {
	for _, title := range []string{"1984 - A Novel", "11/22/63", "2001 - A Space Odyssey"} {
		result, changed := Tidy(TidyInput{Title: title})
		if changed || result.Series != "" || result.SeriesIndex != nil {
			t.Errorf("%q was treated as a volume: %+v", title, result)
		}
	}
}

func TestADistributionTagIsNotPartOfTheTitle(t *testing.T) {
	result, changed := Tidy(TidyInput{Title: "Dune (epub)"})
	if !changed || result.Title != "Dune" {
		t.Fatalf("title = %q changed=%v, want Dune", result.Title, changed)
	}
}

func TestSpacesAndRepeatedNamesAreCollapsed(t *testing.T) {
	result, _ := Tidy(TidyInput{Title: "Dune   -   Messiah"})
	if result.Title != "Dune - Messiah" {
		t.Fatalf("title = %q, want the doubled spaces collapsed", result.Title)
	}

	// No title is cut down at all now — not to its volume, not past a repeated series name. These all stay as they are.
	for _, title := range []string{
		"Dune - Dune Messiah",
		"Harry Potter and the Order of the Phoenix",
		"Totto-chan, the Little Girl at the Window - The Sequel",
	} {
		if kept, changed := Tidy(TidyInput{Title: title, Series: strings.SplitN(title, " ", 2)[0]}); changed {
			t.Errorf("%q was cut to %q", title, kept.Title)
		}
	}
}

func TestAuthorsLoseSeparatorsBlanksAndUnknown(t *testing.T) {
	result, changed := Tidy(TidyInput{Title: "Dune", Authors: []string{"- Asato Asato ", "", "Unknown", "Asato asato"}})
	if !changed {
		t.Fatal("authors were left as they were")
	}
	if len(result.Authors) != 1 || result.Authors[0] != "Asato Asato" {
		t.Fatalf("authors = %v, want one cleaned name", result.Authors)
	}
}

func TestAnAlreadyCleanRowIsLeftAlone(t *testing.T) {
	result, changed := Tidy(TidyInput{
		Title:       "The Last Wish",
		Series:      "The Witcher",
		SeriesIndex: float64Ptr(1),
		Authors:     []string{"Andrzej Sapkowski"},
	})
	if changed {
		t.Fatalf("a clean row was changed: %+v", result)
	}
}

func float64Ptr(value float64) *float64 { return &value }
