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
	matches := func(indexed indexedBook) bool {
		return (request.Series == nil || indexed.book.Series == *request.Series) &&
			(request.Author == nil || bookHasAuthor(indexed.book, *request.Author)) &&
			(bucket == "" || indexed.bucket == bucket) &&
			(query == "" || strings.Contains(indexed.searchKey, query))
	}
	if len(selected) == 1 && request.Series == nil && request.Author == nil && request.Sort == "recent" && query == "" && bucket == "" {
		return selected[0].recentPage(request)
	}
	// A single library without series ordering is already title-sorted by its
	// index. Count the complete result but retain only the requested window.
	if len(selected) == 1 && request.Series == nil && request.Author == nil && request.Sort == "" {
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
			if !matches(indexed) {
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

	items := make([]BookResult, 0)
	for _, library := range selected {
		for _, indexed := range library.snapshot() {
			if matches(indexed) {
				items = append(items, BookResult{LibraryID: library.library.ID, LibraryName: library.library.Name, Book: indexed.book})
			}
		}
	}
	if request.Sort == "series_volume" {
		// Sort the complete result before paging; depth is intentionally ignored
		// here because KOReader presents every volume in numeric order.
		sort.Slice(items, func(i, j int) bool {
			a, b := items[i], items[j]
			x, y := a.Book.SeriesIndex, b.Book.SeriesIndex
			if x == nil || y == nil {
				if x != y {
					return x != nil
				}
			} else if *x != *y {
				return *x < *y
			}
			if a.Book.Title != b.Book.Title {
				return a.Book.Title < b.Book.Title
			}
			if a.LibraryID != b.LibraryID {
				return a.LibraryID < b.LibraryID
			}
			return a.Book.ID < b.Book.ID
		})
	} else if request.Sort == "recent" {
		sort.SliceStable(items, func(i, j int) bool {
			if !items[i].Book.Modified.Equal(items[j].Book.Modified) {
				return items[i].Book.Modified.After(items[j].Book.Modified)
			}
			if items[i].LibraryID != items[j].LibraryID {
				return items[i].LibraryID < items[j].LibraryID
			}
			return items[i].Book.ID < items[j].Book.ID
		})
	} else if request.Sort == "author" || request.Sort == "series" {
		sortBooksByName(items, request.Sort)
	} else if request.Series != nil || len(selected) > 1 {
		sort.SliceStable(items, func(i, j int) bool {
			if request.Series != nil && items[i].LibraryID == items[j].LibraryID {
				if seriesLess(items[i].Book, items[j].Book) {
					return true
				}
				if seriesLess(items[j].Book, items[i].Book) {
					return false
				}
				return items[i].Book.ID < items[j].Book.ID
			}
			left, right := catalog.SortKey(items[i].Book.Title), catalog.SortKey(items[j].Book.Title)
			if left != right {
				return left < right
			}
			if items[i].LibraryID != items[j].LibraryID {
				return items[i].LibraryID < items[j].LibraryID
			}
			return items[i].Book.ID < items[j].Book.ID
		})
	}
	total := len(items)
	anchorFound := request.AfterBookID == ""
	if request.AfterBookID != "" {
		offset = 0
		for index, item := range items {
			if item.LibraryID == request.AfterLibraryID && item.Book.ID == request.AfterBookID {
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
	return BooksResult{Items: items[offset:end], Total: total, AnchorFound: anchorFound, HasMore: end < total}
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
	items := make([]BookResult, 0)
	selected := 0
	for _, library := range s.libraries {
		if libraryID != "" && libraryID != library.library.ID {
			continue
		}
		selected++
		for _, indexed := range library.snapshot() {
			if queryValue == "" || strings.Contains(indexed.searchKey, queryValue) {
				items = append(items, BookResult{LibraryID: library.library.ID, LibraryName: library.library.Name, Book: indexed.book})
			}
		}
	}
	if selected > 1 {
		sort.SliceStable(items, func(i, j int) bool {
			left, right := catalog.SortKey(items[i].Book.Title), catalog.SortKey(items[j].Book.Title)
			if left != right {
				return left < right
			}
			if items[i].LibraryID != items[j].LibraryID {
				return items[i].LibraryID < items[j].LibraryID
			}
			return items[i].Book.ID < items[j].Book.ID
		})
	}
	positions := make(map[string]int)
	counts := make(map[string]int)
	for index, item := range items {
		bucket := catalog.Bucket(item.Book.Title)
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
