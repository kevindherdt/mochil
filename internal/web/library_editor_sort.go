package web

import (
	"bytes"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// editorSortKey is one row's comparable form: the value of the column being sorted and of the series and title every
// order falls back to, folded to lower case once per row instead of on every comparison. Sorting a 5 000-book library
// compares about 60 000 pairs, and the comparator used to lower-case both rows on each of them — a hundred thousand
// short-lived strings, which measured as a third of the request's CPU and drove the garbage collection that followed.
//
// The folded values are slices of one buffer rather than strings of their own: strings.ToLower allocated a string per
// row per column — 10 841 of them, 470 kB and 1.5 ms on the 5 430-book library, for text nothing ever reads as a
// string again. A comparison only needs the bytes.
type editorSortKey struct {
	text    []byte  // the sorted column's text, folded; for the isbn column, the isbn part of the identifiers
	series  []byte  // folded series name, the first fallback
	title   []byte  // folded title, the last fallback
	custom  []byte  // a "#column" value, folded
	number  float64 // a "#column" value that reads as a number
	numeric bool
}

// editorSortKeyOf folds one row's sorted value and its fallback values into arena.
func editorSortKeyOf(row editorRow, column string, arena []byte) (editorSortKey, []byte) {
	key := editorSortKey{}
	arena, key.series = appendFolded(arena, row.Series)
	arena, key.title = appendFolded(arena, row.Title)
	if strings.HasPrefix(column, "#") {
		arena, key.custom = appendFolded(arena, row.Custom[column])
		if number, err := strconv.ParseFloat(row.Custom[column], 64); err == nil {
			key.number, key.numeric = number, true
		}
		return key, arena
	}
	var value string
	switch column {
	case "title":
		key.text = key.title
	case "author":
		value = firstAuthor(row)
	case "path":
		value = row.Path
	case "format":
		value = row.Format
	case "publisher":
		value = row.Publisher
	case "published":
		value = row.Published
	case "language":
		value = row.Language
	case "tags":
		value = strings.Join(row.Tags, ", ")
	case "isbn":
		value = rowISBN(row)
	case "identifiers":
		value = row.Identifiers
	case "title_sort":
		value = row.TitleSort
	case "author_sort":
		value = row.AuthorSort
	case "description":
		value = row.Description
	case "isbn_found":
		value = row.IsbnFound
	case "hardcover":
		value = row.Hardcover
	case "collections":
		value = strings.Join(row.Collections, ", ")
	}
	if value != "" {
		arena, key.text = appendFolded(arena, value)
	}
	return key, arena
}

// appendFolded appends the lower-case form of s to arena and returns the grown arena and the folded bytes inside it. A
// plain ASCII value is folded a byte at a time, which is exactly what strings.ToLower does to it; a value with a byte
// outside ASCII goes through strings.ToLower, so letters such as É keep the case rules the Unicode tables give them.
func appendFolded(arena []byte, s string) ([]byte, []byte) {
	start := len(arena)
	arena = append(arena, s...)
	for i := start; i < len(arena); i++ {
		c := arena[i]
		if c >= utf8.RuneSelf {
			// Not ASCII: the Unicode path, which may fold to a different number of bytes.
			arena = append(arena[:start], strings.ToLower(s)...)
			return arena, arena[start:]
		}
		if 'A' <= c && c <= 'Z' {
			arena[i] = c + 'a' - 'A'
		}
	}
	return arena, arena[start:]
}

func firstAuthor(row editorRow) string {
	if len(row.Authors) == 0 {
		return ""
	}
	return row.Authors[0]
}

// editorRowBlank says whether a row has nothing in the column being sorted. Such a row stays below the others whichever
// way the list is sorted.
func editorRowBlank(row editorRow, column string) bool {
	switch column {
	case "author":
		return firstAuthor(row) == ""
	case "series":
		return row.Series == ""
	case "number":
		return row.SeriesIndex == nil
	case "publisher":
		return row.Publisher == ""
	case "published":
		return row.Published == ""
	case "language":
		return row.Language == ""
	case "tags":
		return len(row.Tags) == 0
	case "rating":
		return row.Rating == 0
	case "words":
		return row.Words == 0
	case "pages":
		return row.Pages == 0
	case "progress":
		return row.Progress == 0
	case "identifiers":
		return row.Identifiers == ""
	case "title_sort":
		return row.TitleSort == ""
	case "author_sort":
		return row.AuthorSort == ""
	case "description":
		return row.Description == ""
	case "isbn_found":
		return row.IsbnFound == ""
	case "match":
		return row.Match == nil
	case "collections":
		return len(row.Collections) == 0
	case "hardcover":
		return row.Hardcover == ""
	}
	return strings.HasPrefix(column, "#") && row.Custom[column] == ""
}

// sortEditorRows orders the list by a column; every order falls back to series, number and title so that equal values
// keep a steady, readable order. Books without a series or number sort last when ascending.
func sortEditorRows(rows []editorRow, column string, descending bool) {
	if len(rows) < 2 {
		return
	}
	custom := strings.HasPrefix(column, "#")
	// One buffer for every folded value. It is sized from the series and title every order folds plus, for a custom
	// column, that column too; the isbn, tags and collections keys fold a value that is not in the row's own fields
	// and grow it by the few kilobytes those columns add.
	size := 0
	for i := range rows {
		size += len(rows[i].Series) + len(rows[i].Title)
		if custom {
			size += len(rows[i].Custom[column])
		}
	}
	arena := make([]byte, 0, size)
	keys := make([]editorSortKey, len(rows))
	blank := make([]bool, len(rows))
	order := make([]int, len(rows))
	for i, row := range rows {
		keys[i], arena = editorSortKeyOf(row, column, arena)
		blank[i], order[i] = editorRowBlank(row, column), i
	}
	// The fallback every column shares: series, then volume, then title.
	bySeries := func(i, j int) int {
		if c := emptyLastCompare(keys[i].series, keys[j].series); c != 0 {
			return c
		}
		if c := numberCompare(rows[i].SeriesIndex, rows[j].SeriesIndex); c != 0 {
			return c
		}
		return bytes.Compare(keys[i].title, keys[j].title)
	}
	compare := func(i, j int) int {
		left, right := keys[i], keys[j]
		if custom {
			if left.numeric && right.numeric && left.number != right.number {
				if left.number < right.number {
					return -1
				}
				return 1
			}
			if c := emptyLastCompare(left.custom, right.custom); c != 0 {
				return c
			}
			return bySeries(i, j)
		}
		switch column {
		case "title", "path", "format":
			if c := bytes.Compare(left.text, right.text); c != 0 {
				return c
			}
		case "author", "publisher", "published", "language", "tags", "identifiers", "title_sort", "author_sort",
			"description", "isbn_found", "hardcover", "collections":
			if c := emptyLastCompare(left.text, right.text); c != 0 {
				return c
			}
		case "isbn":
			// Unlike the other columns, books without one are not kept below: ascending puts the books that have an
			// ISBN first and descending the books that lack one, so either group can be brought to the top.
			if (len(left.text) == 0) != (len(right.text) == 0) {
				return map[bool]int{true: 1, false: -1}[len(left.text) == 0]
			}
			if c := bytes.Compare(left.text, right.text); c != 0 {
				return c
			}
		case "number":
			if c := numberCompare(rows[i].SeriesIndex, rows[j].SeriesIndex); c != 0 {
				return c
			}
		case "match":
			if x, y := matchScoreOf(rows[i]), matchScoreOf(rows[j]); x != y {
				return map[bool]int{true: -1, false: 1}[x < y]
			}
		case "status":
			// The books with the most to look at come last when ascending, first when descending.
			if x, y := len(rows[i].Issues), len(rows[j].Issues); x != y {
				return map[bool]int{true: -1, false: 1}[x < y]
			}
		case "words", "pages":
			x, y := rows[i].Words, rows[j].Words
			if column == "pages" {
				x, y = rows[i].Pages, rows[j].Pages
			}
			if x != y {
				return map[bool]int{true: -1, false: 1}[x < y]
			}
		case "progress":
			if x, y := rows[i].Progress, rows[j].Progress; x != y {
				if x < y {
					return -1
				}
				return 1
			}
		case "rating":
			x, y := rows[i].Rating, rows[j].Rating
			if x != y {
				if x == 0 || y == 0 {
					return map[bool]int{true: 1, false: -1}[x == 0]
				}
				if x < y {
					return -1
				}
				return 1
			}
		}
		return bySeries(i, j)
	}
	slices.SortStableFunc(order, func(i, j int) int {
		if blank[i] != blank[j] {
			if blank[j] {
				return -1
			}
			return 1
		}
		result := compare(i, j)
		if descending {
			return -result
		}
		return result
	})
	applyEditorOrder(rows, order)
}

// applyEditorOrder rearranges rows so that position i holds the row the sorted order names there. It moves each row
// once, chasing one cycle at a time; the version this replaced copied the whole list into a fresh slice and back, which
// on 5 430 books measured 2.5 ms and 2.4 MB of row moves per chunk — as much as everything else the sort does. The
// order is consumed: every position it visits is rewritten to point at itself.
func applyEditorOrder(rows []editorRow, order []int) {
	for start := range rows {
		if order[start] == start {
			continue
		}
		held, position := rows[start], start
		for {
			next := order[position]
			order[position] = position // this position now holds its row; no cycle reaches it again
			if next == start {
				rows[position] = held
				break
			}
			rows[position] = rows[next]
			position = next
		}
	}
}

// emptyLastCompare puts a value that is not there below the ones that are, then compares folded text.
func emptyLastCompare(a, b []byte) int {
	switch {
	case len(a) == 0 && len(b) != 0:
		return 1
	case len(a) != 0 && len(b) == 0:
		return -1
	}
	return bytes.Compare(a, b)
}

// numberCompare puts a book without a volume number below the ones with one.
func numberCompare(a, b *float64) int {
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

// matchScoreOf is how good the best Hardcover match found for a book is; 0 if none was found.
func matchScoreOf(row editorRow) float64 {
	if row.Match == nil {
		return 0
	}
	return row.Match.Score
}

// rowISBN is the isbn entry of a row's identifiers ("goodreads:9, isbn:978…").
func rowISBN(row editorRow) string {
	for _, part := range strings.Split(row.Identifiers, ",") {
		if scheme, value, ok := strings.Cut(strings.TrimSpace(part), ":"); ok && strings.EqualFold(scheme, "isbn") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// editorFieldProblem checks the values the overrides would refuse, so a bad edit is rejected before anything is saved.
func editorFieldProblem(field, value string) string {
	switch field {
	case "rating":
		if value == "" {
			return ""
		}
		if stars, err := strconv.ParseFloat(value, 64); err != nil || stars < 0 || stars > 5 || math.IsNaN(stars) {
			return "A rating is between 0 and 5 stars."
		}
	case "identifiers":
		if value == "" {
			return ""
		}
		for _, part := range strings.Split(value, ",") {
			if scheme, id, ok := strings.Cut(strings.TrimSpace(part), ":"); !ok || strings.TrimSpace(scheme) == "" || strings.TrimSpace(id) == "" || strings.ContainsAny(scheme, " <>&\"'/") {
				return "Identifiers look like isbn:978…, hardcover:dune (separate several with commas)."
			}
		}
	}
	return ""
}
