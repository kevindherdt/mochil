package query

import (
	"path/filepath"
	"sort"
	"strings"

	"mochil/internal/catalog"
)

func (s *Service) Books(request BooksRequest) BooksResult {
	query := strings.ToLower(strings.TrimSpace(request.Query))
	bucket := strings.ToUpper(strings.TrimSpace(request.Bucket))
	limit := request.Limit
	if limit <= 0 {
		limit = 120
	} else if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset := request.Offset
	if offset < 0 {
		offset = 0
	}
	selected := make([]*libraryIndex, 0, len(s.libraries))
	for _, library := range s.libraries {
		if request.LibraryID != "" && request.LibraryID != library.library.ID {
			continue
		}
		selected = append(selected, library)
	}
	matches := func(libraryID string, indexed indexedBook) bool {
		return (request.Series == nil || indexed.book.Series == *request.Series) &&
			(request.Author == nil || bookHasAuthor(indexed.book, *request.Author)) &&
			(bucket == "" || indexed.bucket == bucket) &&
			(query == "" || strings.Contains(indexed.searchKey, query)) &&
			(request.Keep == nil || request.Keep(libraryID, indexed.book))
	}
	// Every fast path below is only valid when nothing else filters the shelf; Keep is a filter, so a status view
	// always takes the general scan. (A fast path that ignores Keep would count and page the whole library.)
	if request.Keep == nil && len(selected) == 1 && request.Series == nil && request.Author == nil && request.Sort == "recent" && query == "" && bucket == "" {
		return selected[0].recentPage(request)
	}
	// A single library without series ordering is already title-sorted by its
	// index. Count the complete result but retain only the requested window.
	if request.Keep == nil && len(selected) == 1 && request.Series == nil && request.Author == nil && request.Sort == "" {
		library := selected[0]
		if query == "" && bucket == "" {
			indexed := library.snapshot()
			start := offset
			anchorFound := request.AfterBookID == ""
			if request.AfterBookID != "" {
				start = 0
				if request.AfterLibraryID == library.library.ID {
					start, anchorFound = library.position(request.AfterBookID)
					start++
				}
			}
			start = min(max(start, 0), len(indexed))
			end := min(start+limit, len(indexed))
			items := make([]BookResult, 0, end-start)
			if anchorFound {
				for _, item := range indexed[start:end] {
					items = append(items, BookResult{LibraryID: library.library.ID, LibraryName: library.library.Name, Book: item.book})
				}
			}
			return BooksResult{Items: items, Total: len(indexed), AnchorFound: anchorFound, HasMore: anchorFound && end < len(indexed)}
		}
		items := make([]BookResult, 0, limit)
		total, start := 0, offset
		anchorFound := request.AfterBookID == ""
		for _, indexed := range library.snapshot() {
			if !matches(library.library.ID, indexed) {
				continue
			}
			if !anchorFound {
				if library.library.ID == request.AfterLibraryID && indexed.book.ID == request.AfterBookID {
					anchorFound = true
					start = total + 1
				}
				total++
				continue
			}
			if total >= start && len(items) < limit {
				items = append(items, BookResult{LibraryID: library.library.ID, LibraryName: library.library.Name, Book: indexed.book})
			}
			total++
		}
		return BooksResult{Items: items, Total: total, AnchorFound: anchorFound, HasMore: anchorFound && start+len(items) < total}
	}

	matched := make([]matchRef, 0, 1024)
	for _, library := range selected {
		books := library.snapshot()
		for index := range books {
			if matches(library.library.ID, books[index]) {
				matched = append(matched, matchRef{library: library, indexed: &books[index]})
			}
		}
	}
	if request.Sort == "series_volume" {
		// Sort the complete result before paging; depth is intentionally ignored
		// here because KOReader presents every volume in numeric order. Two volumes
		// of one series can share an index (or carry none at all), and the reader
		// still expects volume 2 before volume 10.
		sort.Sort(sortedByVolume(matched))
	} else if request.Sort == "recent" {
		sort.SliceStable(matched, func(i, j int) bool {
			if !matched[i].indexed.book.Modified.Equal(matched[j].indexed.book.Modified) {
				return matched[i].indexed.book.Modified.After(matched[j].indexed.book.Modified)
			}
			if matched[i].libraryID() != matched[j].libraryID() {
				return matched[i].libraryID() < matched[j].libraryID()
			}
			return matched[i].indexed.book.ID < matched[j].indexed.book.ID
		})
	} else if request.Sort == "author" || request.Sort == "series" {
		sortBooksByName(matched, request.Sort)
	} else if request.Series != nil {
		sort.Stable(sortedBySeriesTitle(matched))
	} else if len(selected) > 1 {
		// The per-library index already sorts titles naturally; this keeps an
		// all-libraries shelf in the same order one library shows.
		sort.Stable(sortedByTitle(matched))
	}
	total := len(matched)
	anchorFound := request.AfterBookID == ""
	if request.AfterBookID != "" {
		offset = 0
		for index, item := range matched {
			if item.libraryID() == request.AfterLibraryID && item.indexed.book.ID == request.AfterBookID {
				offset, anchorFound = index+1, true
				break
			}
		}
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	items := make([]BookResult, 0, end-offset)
	for _, item := range matched[offset:end] {
		items = append(items, BookResult{LibraryID: item.libraryID(), LibraryName: item.library.library.Name, Book: item.indexed.book})
	}
	return BooksResult{Items: items, Total: total, AnchorFound: anchorFound, HasMore: end < total}
}

func bookHasAuthor(book catalog.Book, wanted string) bool {
	if strings.EqualFold(strings.TrimSpace(wanted), "Unknown author") {
		return len(book.Authors) == 0
	}
	for _, author := range book.Authors {
		if strings.EqualFold(strings.TrimSpace(author), strings.TrimSpace(wanted)) {
			return true
		}
	}
	return false
}

func seriesLess(a, b catalog.Book) bool {
	if a.Series == b.Series {
		leftDepth, rightDepth := strings.Count(filepath.ToSlash(a.Path), "/"), strings.Count(filepath.ToSlash(b.Path), "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
	}
	if a.SeriesIndex == nil && b.SeriesIndex != nil {
		return false
	}
	if a.SeriesIndex != nil && b.SeriesIndex == nil {
		return true
	}
	if a.SeriesIndex != nil && b.SeriesIndex != nil && *a.SeriesIndex != *b.SeriesIndex {
		return *a.SeriesIndex < *b.SeriesIndex
	}
	return catalog.NaturalLess(a.Title, b.Title)
}

// BookNavigation reports the first exact position and total count of each
// title bucket in the same ordering used by Books.
func (s *Service) BookNavigation(libraryID, queryValue string) []BookNavigationResult {
	queryValue = strings.ToLower(strings.TrimSpace(queryValue))
	matched := make([]matchRef, 0)
	selected := 0
	for _, library := range s.libraries {
		if libraryID != "" && libraryID != library.library.ID {
			continue
		}
		selected++
		books := library.snapshot()
		for index := range books {
			if queryValue == "" || strings.Contains(books[index].searchKey, queryValue) {
				matched = append(matched, matchRef{library: library, indexed: &books[index]})
			}
		}
	}
	if selected > 1 {
		// The letter offsets below are read as positions in this same order, so it
		// has to be the order the shelf itself is rendered in.
		sort.Stable(sortedByTitle(matched))
	}
	positions := make(map[string]int)
	counts := make(map[string]int)
	for index, item := range matched {
		bucket := catalog.Bucket(item.indexed.book.Title)
		if counts[bucket] == 0 {
			positions[bucket] = index
		}
		counts[bucket]++
	}
	out := make([]BookNavigationResult, 0, len(counts))
	for _, character := range "#ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		bucket := string(character)
		if counts[bucket] > 0 {
			out = append(out, BookNavigationResult{Bucket: bucket, Offset: positions[bucket], Count: counts[bucket]})
		}
	}
	return out
}

func (i *libraryIndex) recentPage(request BooksRequest) BooksResult {
	_ = i.snapshot()
	i.mu.RLock()
	defer i.mu.RUnlock()
	start := max(request.Offset, 0)
	anchorFound := request.AfterBookID == ""
	if request.AfterBookID != "" && request.AfterLibraryID == i.library.ID {
		if position, ok := i.recentPositions[request.AfterBookID]; ok {
			start, anchorFound = position+1, true
		}
	}
	start = min(start, len(i.recent))
	limit := request.Limit
	if limit <= 0 {
		limit = 120
	} else if limit > MaxPageSize {
		limit = MaxPageSize
	}
	end := min(start+limit, len(i.recent))
	if !anchorFound {
		end = start
	}
	items := make([]BookResult, 0, end-start)
	for _, index := range i.recent[start:end] {
		item := i.books[index]
		items = append(items, BookResult{LibraryID: i.library.ID, LibraryName: i.library.Name, Book: item.book})
	}
	return BooksResult{Items: items, Total: len(i.recent), AnchorFound: anchorFound, HasMore: anchorFound && end < len(i.recent)}
}

func (i *libraryIndex) position(bookID string) (int, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	position, ok := i.positions[bookID]
	return position, ok
}
