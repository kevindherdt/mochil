package query

import (
	"sort"
	"strings"

	"mochil/internal/catalog"
)

// A shelf that spans libraries is sorted on every request, unlike one library's
// index which is sorted once when it is built. Two costs were paid per request
// there, both proportional to the whole shelf rather than the page: every match
// was copied into a BookResult before sorting (50k books, 424 bytes each, for a
// 500-book page), and the title keys were normalized inside the comparator, once
// per comparison instead of once per book. These sorters carry each key beside
// a reference to its book and swap the two together, so a request pays for what
// it filters and pages, not for what it copies.
type matchRef struct {
	library *libraryIndex
	indexed *indexedBook
}

func (m matchRef) libraryID() string { return m.library.library.ID }

func titleKeys(matched []matchRef) []string {
	keys := make([]string, len(matched))
	for index := range matched {
		keys[index] = catalog.SortKey(matched[index].indexed.book.Title)
	}
	return keys
}

// keyedTitleOrder orders by normalized title, then library and id: the same
// books come out in the same order whichever shelf asks.
type keyedTitleOrder struct {
	matched []matchRef
	keys    []string
}

func sortedByTitle(matched []matchRef) keyedTitleOrder {
	return keyedTitleOrder{matched: matched, keys: titleKeys(matched)}
}

func (o keyedTitleOrder) Len() int { return len(o.matched) }

func (o keyedTitleOrder) Swap(i, j int) {
	o.matched[i], o.matched[j] = o.matched[j], o.matched[i]
	o.keys[i], o.keys[j] = o.keys[j], o.keys[i]
}

func (o keyedTitleOrder) Less(i, j int) bool {
	if o.keys[i] != o.keys[j] {
		return catalog.NaturalKeyLess(o.keys[i], o.keys[j])
	}
	if left, right := o.matched[i].libraryID(), o.matched[j].libraryID(); left != right {
		return left < right
	}
	return o.matched[i].indexed.book.ID < o.matched[j].indexed.book.ID
}

// keyedSeriesTitleOrder is a series shelf: books of one library follow their
// series position, books of different libraries their title.
type keyedSeriesTitleOrder struct {
	matched []matchRef
	keys    []string
}

func sortedBySeriesTitle(matched []matchRef) keyedSeriesTitleOrder {
	return keyedSeriesTitleOrder{matched: matched, keys: titleKeys(matched)}
}

func (o keyedSeriesTitleOrder) Len() int { return len(o.matched) }

func (o keyedSeriesTitleOrder) Swap(i, j int) {
	o.matched[i], o.matched[j] = o.matched[j], o.matched[i]
	o.keys[i], o.keys[j] = o.keys[j], o.keys[i]
}

func (o keyedSeriesTitleOrder) Less(i, j int) bool {
	a, b := o.matched[i], o.matched[j]
	if a.libraryID() == b.libraryID() {
		if seriesLess(a.indexed.book, b.indexed.book) {
			return true
		}
		if seriesLess(b.indexed.book, a.indexed.book) {
			return false
		}
		return a.indexed.book.ID < b.indexed.book.ID
	}
	if o.keys[i] != o.keys[j] {
		return catalog.NaturalKeyLess(o.keys[i], o.keys[j])
	}
	return a.libraryID() < b.libraryID()
}

// keyedVolumeOrder is the series view: volume numbers first, then the same title
// order and tie-breaks as keyedTitleOrder.
type keyedVolumeOrder struct {
	matched []matchRef
	keys    []string
}

func sortedByVolume(matched []matchRef) keyedVolumeOrder {
	return keyedVolumeOrder{matched: matched, keys: titleKeys(matched)}
}

func (o keyedVolumeOrder) Len() int { return len(o.matched) }

func (o keyedVolumeOrder) Swap(i, j int) {
	o.matched[i], o.matched[j] = o.matched[j], o.matched[i]
	o.keys[i], o.keys[j] = o.keys[j], o.keys[i]
}

func (o keyedVolumeOrder) Less(i, j int) bool {
	a, b := o.matched[i], o.matched[j]
	x, y := a.indexed.book.SeriesIndex, b.indexed.book.SeriesIndex
	if x == nil || y == nil {
		if x != y {
			return x != nil
		}
	} else if *x != *y {
		return *x < *y
	}
	if o.keys[i] != o.keys[j] {
		return catalog.NaturalKeyLess(o.keys[i], o.keys[j])
	}
	if left, right := a.libraryID(), b.libraryID(); left != right {
		return left < right
	}
	return a.indexed.book.ID < b.indexed.book.ID
}

// keyedNameOrder orders a result by author or by series. Books with nothing in
// the chosen field come last, and equal values fall back to series order and
// then title so the list stays steady.
type keyedNameOrder struct {
	matched []matchRef
	authors []string
	series  []string
	titles  []string
	name    string
}

func sortBooksByName(matched []matchRef, name string) {
	if len(matched) < 2 {
		return
	}
	order := keyedNameOrder{
		matched: matched,
		authors: make([]string, len(matched)),
		series:  make([]string, len(matched)),
		titles:  titleKeys(matched),
		name:    name,
	}
	for index, item := range matched {
		order.authors[index] = authorKey(item.indexed.book)
		order.series[index] = catalog.SortKey(item.indexed.book.Series)
	}
	sort.Stable(order)
}

func authorKey(book catalog.Book) string {
	if len(book.AuthorSort) > 0 && strings.TrimSpace(book.AuthorSort[0]) != "" {
		return strings.ToLower(strings.TrimSpace(book.AuthorSort[0]))
	}
	if len(book.Authors) > 0 {
		return catalog.SortKey(book.Authors[0])
	}
	return ""
}

func (o keyedNameOrder) Len() int { return len(o.matched) }

func (o keyedNameOrder) Swap(i, j int) {
	o.matched[i], o.matched[j] = o.matched[j], o.matched[i]
	o.authors[i], o.authors[j] = o.authors[j], o.authors[i]
	o.series[i], o.series[j] = o.series[j], o.series[i]
	o.titles[i], o.titles[j] = o.titles[j], o.titles[i]
}

func (o keyedNameOrder) Less(i, j int) bool {
	a, b := o.matched[i], o.matched[j]
	left, right := o.authors[i], o.authors[j]
	if o.name == "series" {
		left, right = o.series[i], o.series[j]
	}
	if (left == "") != (right == "") {
		return right == ""
	}
	if left != right {
		// The keys are already normalized, so this only has to treat digit runs as
		// numbers: an author or series called "2" belongs before "10".
		return catalog.NaturalKeyLess(left, right)
	}
	if o.name == "author" {
		if sa, sb := o.series[i], o.series[j]; sa != sb {
			if sa == "" || sb == "" {
				return sb == ""
			}
			return catalog.NaturalKeyLess(sa, sb)
		}
	}
	if bookA, bookB := a.indexed.book, b.indexed.book; bookA.Series == bookB.Series && bookA.Series != "" {
		if seriesLess(bookA, bookB) {
			return true
		}
		if seriesLess(bookB, bookA) {
			return false
		}
	} else if catalog.NaturalKeyLess(o.titles[i], o.titles[j]) || catalog.NaturalKeyLess(o.titles[j], o.titles[i]) {
		return catalog.NaturalKeyLess(o.titles[i], o.titles[j])
	}
	if left, right := a.libraryID(), b.libraryID(); left != right {
		return left < right
	}
	return a.indexed.book.ID < b.indexed.book.ID
}
