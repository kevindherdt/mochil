package web

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// sortEditorRows orders the list by a column; every order falls back to series, number and title so that equal values
// keep a steady, readable order. Books without a series or number sort last when ascending.
func sortEditorRows(rows []editorRow, column string, descending bool) {
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
			// Unlike the other columns, books without one are not kept below: ascending puts the books that have an
			// ISBN first and descending the books that lack one, so either group can be brought to the top.
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
			// The books with the most to look at come last when ascending, first when descending.
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
	// A book with nothing in the chosen column stays below the others, whichever way the list is sorted.
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
