package web

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// sortEditorRowsBefore is the comparator as it stood before the sort keys were precomputed. It stays here as the
// reference the faster version has to agree with, row for row, on every column and in both directions.
func sortEditorRowsBefore(rows []editorRow, column string, descending bool) {
	text := func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }
	emptyLast := func(a, b string) int {
		switch {
		case a == "" && b != "":
			return 1
		case a != "" && b == "":
			return -1
		}
		return text(a, b)
	}
	number := func(a, b *float64) int {
		switch {
		case a == nil && b != nil:
			return 1
		case a != nil && b == nil:
			return -1
		case a == nil || *a == *b:
			return 0
		case *a < *b:
			return -1
		}
		return 1
	}
	author := func(r editorRow) string {
		if len(r.Authors) == 0 {
			return ""
		}
		return r.Authors[0]
	}
	bySeries := func(a, b editorRow) int {
		if c := emptyLast(a.Series, b.Series); c != 0 {
			return c
		}
		if c := number(a.SeriesIndex, b.SeriesIndex); c != 0 {
			return c
		}
		return text(a.Title, b.Title)
	}
	compare := func(a, b editorRow) int {
		if strings.HasPrefix(column, "#") {
			x, y := a.Custom[column], b.Custom[column]
			nx, errX := strconv.ParseFloat(x, 64)
			ny, errY := strconv.ParseFloat(y, 64)
			if errX == nil && errY == nil && nx != ny {
				if nx < ny {
					return -1
				}
				return 1
			}
			if c := emptyLast(x, y); c != 0 {
				return c
			}
			return bySeries(a, b)
		}
		switch column {
		case "title":
			if c := text(a.Title, b.Title); c != 0 {
				return c
			}
		case "author":
			if c := emptyLast(author(a), author(b)); c != 0 {
				return c
			}
		case "number":
			if c := number(a.SeriesIndex, b.SeriesIndex); c != 0 {
				return c
			}
		case "path":
			if c := text(a.Path, b.Path); c != 0 {
				return c
			}
		case "format":
			if c := text(a.Format, b.Format); c != 0 {
				return c
			}
		case "publisher":
			if c := emptyLast(a.Publisher, b.Publisher); c != 0 {
				return c
			}
		case "published":
			if c := emptyLast(a.Published, b.Published); c != 0 {
				return c
			}
		case "language":
			if c := emptyLast(a.Language, b.Language); c != 0 {
				return c
			}
		case "tags":
			if c := emptyLast(strings.Join(a.Tags, ", "), strings.Join(b.Tags, ", ")); c != 0 {
				return c
			}
		case "isbn":
			x, y := rowISBN(a), rowISBN(b)
			if (x == "") != (y == "") {
				return map[bool]int{true: 1, false: -1}[x == ""]
			}
			if c := text(x, y); c != 0 {
				return c
			}
		case "identifiers":
			if c := emptyLast(a.Identifiers, b.Identifiers); c != 0 {
				return c
			}
		case "title_sort":
			if c := emptyLast(a.TitleSort, b.TitleSort); c != 0 {
				return c
			}
		case "author_sort":
			if c := emptyLast(a.AuthorSort, b.AuthorSort); c != 0 {
				return c
			}
		case "description":
			if c := emptyLast(a.Description, b.Description); c != 0 {
				return c
			}
		case "isbn_found":
			if c := emptyLast(a.IsbnFound, b.IsbnFound); c != 0 {
				return c
			}
		case "match":
			if x, y := matchScoreOf(a), matchScoreOf(b); x != y {
				return map[bool]int{true: -1, false: 1}[x < y]
			}
		case "hardcover":
			if c := emptyLast(a.Hardcover, b.Hardcover); c != 0 {
				return c
			}
		case "collections":
			if c := emptyLast(strings.Join(a.Collections, ", "), strings.Join(b.Collections, ", ")); c != 0 {
				return c
			}
		case "status":
			if x, y := len(a.Issues), len(b.Issues); x != y {
				return map[bool]int{true: -1, false: 1}[x < y]
			}
		case "words":
			if a.Words != b.Words {
				return map[bool]int{true: -1, false: 1}[a.Words < b.Words]
			}
		case "pages":
			if a.Pages != b.Pages {
				return map[bool]int{true: -1, false: 1}[a.Pages < b.Pages]
			}
		case "progress":
			if a.Progress != b.Progress {
				if a.Progress < b.Progress {
					return -1
				}
				return 1
			}
		case "rating":
			if a.Rating != b.Rating {
				if a.Rating == 0 || b.Rating == 0 {
					return map[bool]int{true: 1, false: -1}[a.Rating == 0]
				}
				if a.Rating < b.Rating {
					return -1
				}
				return 1
			}
		}
		return bySeries(a, b)
	}
	empty := func(r editorRow) bool {
		switch column {
		case "author":
			return author(r) == ""
		case "series":
			return r.Series == ""
		case "number":
			return r.SeriesIndex == nil
		case "publisher":
			return r.Publisher == ""
		case "published":
			return r.Published == ""
		case "language":
			return r.Language == ""
		case "tags":
			return len(r.Tags) == 0
		case "rating":
			return r.Rating == 0
		case "words":
			return r.Words == 0
		case "pages":
			return r.Pages == 0
		case "progress":
			return r.Progress == 0
		case "identifiers":
			return r.Identifiers == ""
		case "title_sort":
			return r.TitleSort == ""
		case "author_sort":
			return r.AuthorSort == ""
		case "description":
			return r.Description == ""
		case "isbn_found":
			return r.IsbnFound == ""
		case "match":
			return r.Match == nil
		case "collections":
			return len(r.Collections) == 0
		case "hardcover":
			return r.Hardcover == ""
		}
		return strings.HasPrefix(column, "#") && r.Custom[column] == ""
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if ei, ej := empty(rows[i]), empty(rows[j]); ei != ej {
			return ej
		}
		if descending {
			return compare(rows[i], rows[j]) > 0
		}
		return compare(rows[i], rows[j]) < 0
	})
}

// The new sort has to order every column exactly as the old comparator did, including its tie-breaks and the rule that
// a book with nothing in the sorted column stays last in both directions.
func TestSortingAWholeLibraryAgreesWithTheOldComparator(t *testing.T) {
	random := rand.New(rand.NewSource(7))
	caseOf := func(value string) string {
		switch random.Intn(3) {
		case 0:
			return strings.ToUpper(value)
		case 1:
			return strings.ToLower(value)
		}
		return value
	}
	text := func(words ...string) string {
		if random.Intn(4) == 0 {
			return ""
		}
		return caseOf(words[random.Intn(len(words))])
	}
	rows := make([]editorRow, 300)
	for i := range rows {
		row := editorRow{
			ID:          "b" + strconv.Itoa(i),
			Title:       text("dune", "Neuromancer", "a", "the long way", "Émile"),
			Series:      text("dune", "Other", "series"),
			Publisher:   text("Ace", "tor"),
			Published:   text("1965", "2019"),
			Language:    text("en", "fr"),
			Path:        text("a/b.epub", "z/a.epub"),
			Format:      text("epub", "cbz"),
			Identifiers: text("isbn:9782", "isbn:9781", "goodreads:4"),
			TitleSort:   text("dune", "neuromancer"),
			AuthorSort:  text("Herbert, Frank", "Gibson, William"),
			Description: text("Spice.", "Cyberspace."),
			IsbnFound:   text("2222", "1111"),
			Hardcover:   text("synced", "doubt"),
			Progress:    float64(random.Intn(4)) / 4,
			Rating:      float64(random.Intn(6)),
			Words:       random.Intn(3) * 1000,
			Pages:       random.Intn(3) * 100,
			Custom:      map[string]string{"#score": []string{"1", "10", "", "abc", "2.5"}[random.Intn(5)]},
		}
		if random.Intn(3) > 0 {
			row.Authors = []string{text("Herbert", "Gibson", "Ōe")}
		}
		if random.Intn(3) > 0 {
			row.Tags = []string{text("sci-fi", "Space", "owned")}
		}
		if random.Intn(3) > 0 {
			row.Collections = []string{text("Favourites", "to read")}
		}
		if random.Intn(4) > 0 {
			row.SeriesIndex = ptr(float64(random.Intn(4)))
		}
		if random.Intn(5) > 0 {
			row.Match = &matchCandidate{Score: float64(random.Intn(3)) / 2}
		}
		for n := random.Intn(3); n > 0; n-- {
			row.Issues = append(row.Issues, "issue")
		}
		rows[i] = row
	}
	columns := []string{"", "title", "author", "series", "number", "path", "format", "publisher", "published",
		"language", "tags", "isbn", "identifiers", "title_sort", "author_sort", "description", "isbn_found", "match",
		"hardcover", "collections", "status", "words", "pages", "progress", "rating", "#score", "nonsense"}
	for _, column := range columns {
		for _, descending := range []bool{false, true} {
			want := append([]editorRow(nil), rows...)
			sortEditorRowsBefore(want, column, descending)
			got := append([]editorRow(nil), rows...)
			sortEditorRows(got, column, descending)
			for i := range want {
				if got[i].ID != want[i].ID {
					t.Fatalf("column %q descending %v: position %d is %s, want %s\n got %v\nwant %v",
						column, descending, i, got[i].ID, want[i].ID, ids(got), ids(want))
				}
			}
		}
	}
}

func ids(rows []editorRow) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		out[i] = row.ID
	}
	return out
}

// Folding a value has to read exactly like strings.ToLower would, because the sort's order is that order: ASCII case
// is folded a byte at a time into the shared buffer, and a value with a byte outside ASCII takes the Unicode path.
func TestFoldingAValueReadsLikeToLower(t *testing.T) {
	values := []string{"Dune", "DUNE", "dune", "Émile Zola", "Ōe no Masafu", "İstanbul", "ß", "", "a\x00B", "\xffabc",
		"MiXeD Case", strings.Repeat("A Title With Capitals ", 8)}
	arena := make([]byte, 0, 16)
	folded := make([]string, 0, 2*len(values))
	for _, value := range values {
		var slot []byte
		arena, slot = appendFolded(arena, value)
		folded = append(folded, string(slot))
		// Twice each, so a value appended after it could only be read wrong if the buffer were reused in place.
		arena, slot = appendFolded(arena, value)
		folded = append(folded, string(slot))
	}
	for i, value := range values {
		want := strings.ToLower(value)
		if got := folded[2*i]; got != want {
			t.Fatalf("%q folded to %q, want %q", value, got, want)
		}
		if got := folded[2*i+1]; got != want {
			t.Fatalf("%q folded to %q on the second append, want %q", value, got, want)
		}
	}
}

// A plain ASCII value is folded without allocating: this is what the sort gained, and a regression would put a short
// string back on the heap for every row of every chunk.
func TestFoldingAnASCIIValueDoesNotAllocate(t *testing.T) {
	arena := make([]byte, 0, 256)
	allocs := testing.AllocsPerRun(100, func() {
		arena, _ = appendFolded(arena[:0], "A Mixed Case Series")
	})
	if allocs != 0 {
		t.Fatalf("folding an ASCII value allocated %v times, want 0", allocs)
	}
	if got := string(arena); got != "a mixed case series" {
		t.Fatalf("folded to %q", got)
	}
}

// The sort order is applied to the rows in place, one cycle at a time. This is the half of that which the reference
// comparison in TestSortingAWholeLibraryAgreesWithTheOldComparator cannot pin down on its own: that every position
// ends up holding the row the order names, for orders made of several cycles, of one cycle, and of none at all.
func TestApplyingASortOrderInPlaceMovesEveryRowOnce(t *testing.T) {
	random := rand.New(rand.NewSource(11))
	for trial := 0; trial < 200; trial++ {
		size := 1 + random.Intn(40)
		rows := make([]editorRow, size)
		for i := range rows {
			rows[i].ID = strconv.Itoa(i)
		}
		order := random.Perm(size)
		want := make([]editorRow, size)
		for position, index := range order {
			want[position] = rows[index]
		}
		applyEditorOrder(rows, append([]int(nil), order...))
		for i := range rows {
			if rows[i].ID != want[i].ID {
				t.Fatalf("trial %d: position %d holds %s, want %s", trial, i, rows[i].ID, want[i].ID)
			}
		}
	}
}
