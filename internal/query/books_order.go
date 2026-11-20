package query

import (
	"sort"
	"strings"

	"mochil/internal/catalog"
)

// sortBooksByName orders a result by author or by series. Books with nothing in the chosen field come last, and equal
// values fall back to series order and then title so the list stays steady.
func sortBooksByName(items []BookResult, name string) {
	authorKey := func(book catalog.Book) string {
		if len(book.AuthorSort) > 0 && strings.TrimSpace(book.AuthorSort[0]) != "" {
			return strings.ToLower(strings.TrimSpace(book.AuthorSort[0]))
		}
		if len(book.Authors) > 0 {
			return catalog.SortKey(book.Authors[0])
		}
		return ""
	}
	seriesKey := func(book catalog.Book) string { return catalog.SortKey(book.Series) }
	key := authorKey
	if name == "series" {
		key = seriesKey
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i].Book, items[j].Book
		left, right := key(a), key(b)
		if (left == "") != (right == "") {
			return right == ""
		}
		if left != right {
			return left < right
		}
		if name == "author" {
			if sa, sb := seriesKey(a), seriesKey(b); sa != sb {
				if sa == "" || sb == "" {
					return sb == ""
				}
				return sa < sb
			}
		}
		if a.Series == b.Series && a.Series != "" {
			if seriesLess(a, b) {
				return true
			}
			if seriesLess(b, a) {
				return false
			}
		} else if title := catalog.NaturalLess(a.Title, b.Title); title || catalog.NaturalLess(b.Title, a.Title) {
			return title
		}
		if items[i].LibraryID != items[j].LibraryID {
			return items[i].LibraryID < items[j].LibraryID
		}
		return a.ID < b.ID
	})
}
