package booktext

import (
	"strings"
	"testing"
	"time"
)

// A chapter that arrives as one text block (a <div> full of <br/>, or a <pre>) used to be quadratic: the abbreviation and
// initials checks were handed the whole prefix for every full stop, so 400KB took 92 seconds and 100KB took 5.8s. The
// window is bounded now; this keeps it bounded.
func TestSentencesStayLinearOnASingleBlockChapter(t *testing.T) {
	piece := "The sea was calm tonight and the stars were out over the whole quiet harbour. "
	text := strings.Repeat(piece, 400*1024/len(piece)) // about 400KB, one block
	started := time.Now()
	sentences := spans([]rune(text))
	took := time.Since(started)
	if len(sentences) < 100 {
		t.Fatalf("only %d sentences out of %d bytes", len(sentences), len(text))
	}
	if took > 3*time.Second {
		t.Fatalf("splitting %d bytes took %s; the prefix scan is back", len(text), took)
	}
}

// The window must not change what counts as an abbreviation or as initials, which is why the patterns are end-anchored.
func TestTrailingTokenKeepsAbbreviationsAndInitials(t *testing.T) {
	for _, test := range []struct {
		text   string
		cut    int
		abbrev bool
	}{
		{"said Dr", 7, true},
		{"said Mevr", 9, true},
		{"written by J.R.R", 16, true},
		{"a plain word here", 17, false},
		{"", 0, false},
	} {
		window := trailingToken(test.text, test.cut, 32)
		if got := abbreviation.MatchString(window) || initials.MatchString(window); got != test.abbrev {
			t.Fatalf("%q window %q: abbreviation/initials = %v, want %v", test.text, window, got, test.abbrev)
		}
	}
}
