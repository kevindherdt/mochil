package web

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

// TestDuplicateVolumesGroupsTheWayTheStringKeyDid pins the equality the duplicate scan has to keep now that it
// compares folded series bytes and volume bits instead of a built "series\x00number" string: two books belong to the
// same group exactly when that string would have been the same for both.
func TestDuplicateVolumesGroupsTheWayTheStringKeyDid(t *testing.T) {
	numbered := func(id, series string, index float64) catalog.Book {
		return catalog.Book{ID: id, Title: "Book " + id, Authors: []string{"Author"}, Series: series, SeriesIndex: &index}
	}
	for _, test := range []struct {
		name  string
		books []catalog.Book
		want  [][]string
	}{
		{
			name:  "series folded, volume equal",
			books: []catalog.Book{numbered("a", "Dune", 1), numbered("b", "DUNE", 1), numbered("c", "Dune", 2)},
			want:  [][]string{{"duplicate"}, {"duplicate"}, nil},
		},
		{
			name:  "the same volume written differently is the same number",
			books: []catalog.Book{numbered("a", "Dune", 1), numbered("b", "Dune", 1.0)},
			want:  [][]string{{"duplicate"}, {"duplicate"}},
		},
		{
			// Both formatted as "NaN", so they used to share a key and still must — even when the two NaNs are
			// different bit patterns, which no equality test on the value would call equal.
			name: "two unknown volumes",
			books: []catalog.Book{
				numbered("a", "Dune", math.Float64frombits(0x7FF8000000000001)),
				numbered("b", "Dune", math.Float64frombits(0x7FF8000000000ABC)),
			},
			want: [][]string{{"duplicate"}, {"duplicate"}},
		},
		{
			// They formatted as "-0" and "0": two different volumes, not one book listed twice.
			name:  "negative zero is not zero",
			books: []catalog.Book{numbered("a", "Dune", 0), numbered("b", "Dune", math.Copysign(0, -1))},
			want:  [][]string{nil, nil},
		},
		{
			name: "non-ASCII series name folded",
			books: []catalog.Book{
				numbered("a", "Été", 1), numbered("b", "ÉTÉ", 1), numbered("c", "été", 2), numbered("d", "Été", 2),
			},
			want: [][]string{{"duplicate"}, {"duplicate"}, {"duplicate"}, {"duplicate"}},
		},
		{
			name: "a book without a series or number never duplicates",
			books: []catalog.Book{
				{ID: "a", Title: "Loose", Authors: []string{"A"}, Series: "Dune"},
				{ID: "b", Title: "Loose again", Authors: []string{"A"}, Series: "Dune"},
				{ID: "c", Title: "Different series", Authors: []string{"A"}, Series: "Other", SeriesIndex: ptr(1)},
			},
			want: [][]string{{"no_number"}, {"no_number"}, nil},
		},
		{
			name:  "three books in one slot are all flagged",
			books: []catalog.Book{numbered("a", "Dune", 3), numbered("b", "dune", 3), numbered("c", "Dune", 3)},
			want:  [][]string{{"duplicate"}, {"duplicate"}, {"duplicate"}},
		},
		{
			name:  "the order of a book's own issues is unchanged",
			books: []catalog.Book{{ID: "a", Title: "A Side Story", Series: "Dune", SeriesIndex: ptr(1)}, numbered("b", "Dune", 1)},
			want:  [][]string{{"duplicate", "no_author", "side_story"}, {"duplicate"}},
		},
	} {
		if got := editorIssues(test.books); !equalIssues(got, test.want) {
			t.Errorf("%s: issues = %v, want %v", test.name, got, test.want)
		}
	}
}

// equalIssues compares issue lists, treating a missing entry and an empty one as the same.
func equalIssues(got, want [][]string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if len(got[i]) != len(want[i]) {
			return false
		}
		for j := range got[i] {
			if got[i][j] != want[i][j] {
				return false
			}
		}
	}
	return true
}

// TestContainsLoweredAnswersWhatLoweringWould checks the allocation-free scan against the function it replaced, over
// text it can and cannot take its fast path on, including a byte that is not valid UTF-8.
func TestContainsLoweredAnswersWhatLoweringWould(t *testing.T) {
	for _, title := range []string{
		"", "Side Story", "SIDE STORY", "a side story of sorts", "side stor", "s", "Sidestory",
		"Dune", "The É side story", "Ōe side story", "İ side story", "ǅ side story", "side stor\xff",
		"\xff\xfe side story", "SİDE STORY", "sİde story", "SideStory side storY",
	} {
		for _, word := range []string{"side stor", "side story", "dune", "é", "i", "", "story"} {
			want := strings.Contains(strings.ToLower(title), word)
			if got := containsLowered(title, word); got != want {
				t.Errorf("containsLowered(%q, %q) = %v, lowering says %v", title, word, got, want)
			}
		}
	}
}

// TestEditorIssuesAllocatesPerBookItDoesNotNeed is the cost this pass removed: a lowered series name and a built
// volume string for every book in the library, six times over to fill the editor. The scan may allocate its two
// buffers, but nothing per book.
func TestEditorIssuesAllocatesPerBookItDoesNotNeed(t *testing.T) {
	books := make([]catalog.Book, 400)
	for i := range books {
		index := float64(i)
		books[i] = catalog.Book{ID: fmt.Sprint(i), Title: fmt.Sprintf("Book %d", i), Authors: []string{"A"}, Series: fmt.Sprintf("Series %d", i), SeriesIndex: &index}
	}
	many := testing.AllocsPerRun(20, func() {
		editorIssues(books)
	})
	few := testing.AllocsPerRun(20, func() {
		editorIssues(books[:40])
	})
	if many-few > 4 {
		t.Fatalf("%v allocations for 400 books against %v for 40: the scan allocates per book", many, few)
	}
}
