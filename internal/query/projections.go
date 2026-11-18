package query

import (
	"sort"
	"strings"

	"mochil/internal/catalog"
)

type SeriesRequest struct {
	LibraryID      string
	Query          string
	Bucket         string
	Limit          int
	Offset         int
	AfterLibraryID string
	AfterID        string
}

type SeriesResult struct {
	LibraryID      string
	LibraryName    string
	Name           string
	Bucket         string
	Count          int
	Unsorted       bool
	Representative catalog.Book
}

type SeriesPage struct {
	Items       []SeriesResult
	Total       int
	AnchorFound bool
	HasMore     bool
}

type AuthorResult struct {
	LibraryID      string
	LibraryName    string
	Name           string
	Bucket         string
	Count          int
	Unknown        bool
	Representative catalog.Book
}

type AuthorPage struct {
	Items       []AuthorResult
	Total       int
	AnchorFound bool
	HasMore     bool
}

func (s *Service) Series(request SeriesRequest) SeriesPage {
	query := strings.ToLower(strings.TrimSpace(request.Query))
	bucket := strings.ToUpper(strings.TrimSpace(request.Bucket))
	if query == "" && bucket == "" {
		if request.LibraryID != "" {
			for _, library := range s.libraries {
				if request.LibraryID == library.library.ID {
					return library.seriesPage(request)
				}
			}
			return SeriesPage{AnchorFound: request.AfterID == ""}
		}
		out := make([]SeriesResult, 0)
		for _, library := range s.libraries {
			if request.LibraryID == "" || request.LibraryID == library.library.ID {
				out = append(out, library.seriesSnapshot()...)
			}
		}
		sortSeriesResults(out)
		return paginateSeries(out, request)
	}
	type seriesKey struct{ libraryID, name string }
	counts := make(map[seriesKey]int)
	representatives := make(map[seriesKey]indexedBook)
	standaloneCounts := make(map[string]int)
	standaloneRepresentatives := make(map[string]indexedBook)
	libraryNames := make(map[string]string)
	for _, library := range s.libraries {
		if request.LibraryID != "" && request.LibraryID != library.library.ID {
			continue
		}
		libraryNames[library.library.ID] = library.library.Name
		for _, indexed := range library.snapshot() {
			book := indexed.book
			if book.Series == "" {
				if (query == "" || strings.Contains(indexed.searchKey, query)) && (bucket == "" || indexed.bucket == bucket) {
					standaloneCounts[library.library.ID]++
					if current, ok := standaloneRepresentatives[library.library.ID]; !ok || catalog.NaturalKeyLess(indexed.titleKey, current.titleKey) || (indexed.titleKey == current.titleKey && book.ID < current.book.ID) {
						standaloneRepresentatives[library.library.ID] = indexed
					}
				}
				continue
			}
			if query == "" || strings.Contains(indexed.searchKey, query) {
				key := seriesKey{library.library.ID, book.Series}
				counts[key]++
				if current, ok := representatives[key]; !ok || indexedSeriesLess(indexed, current) || (!indexedSeriesLess(current, indexed) && book.ID < current.book.ID) {
					representatives[key] = indexed
				}
			}
		}
	}
	out := make([]SeriesResult, 0, len(counts)+len(standaloneCounts))
	for key, count := range counts {
		seriesBucket := catalog.Bucket(key.name)
		if bucket == "" || bucket == seriesBucket {
			out = append(out, SeriesResult{LibraryID: key.libraryID, LibraryName: libraryNames[key.libraryID], Name: key.name, Bucket: seriesBucket, Count: count, Representative: representatives[key].book})
		}
	}
	for libraryID, count := range standaloneCounts {
		if count > 0 {
			out = append(out, SeriesResult{LibraryID: libraryID, LibraryName: libraryNames[libraryID], Name: "Standalone books", Count: count, Unsorted: true, Representative: standaloneRepresentatives[libraryID].book})
		}
	}
	sortSeriesResults(out)
	return paginateSeries(out, request)
}

func paginateSeries(out []SeriesResult, request SeriesRequest) SeriesPage {
	total := len(out)
	if request.Limit <= 0 {
		return SeriesPage{Items: out, Total: total, AnchorFound: request.AfterID == "", HasMore: false}
	}
	limit := request.Limit
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	offset := request.Offset
	anchorFound := request.AfterID == ""
	if request.AfterID != "" {
		offset = 0
		for index, item := range out {
			if item.LibraryID == request.AfterLibraryID && item.Representative.ID == request.AfterID {
				offset, anchorFound = index+1, true
				break
			}
		}
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return SeriesPage{Items: out[offset:end], Total: total, AnchorFound: anchorFound, HasMore: end < total}
}

func sortSeriesResults(out []SeriesResult) {
	keys := make(map[string]string, len(out))
	for _, item := range out {
		if _, exists := keys[item.Name]; !exists {
			keys[item.Name] = catalog.SortKey(item.Name)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Unsorted != out[j].Unsorted {
			return !out[i].Unsorted
		}
		left, right := keys[out[i].Name], keys[out[j].Name]
		if left != right {
			return catalog.NaturalKeyLess(left, right)
		}
		return out[i].LibraryID < out[j].LibraryID
	})
}

func (s *Service) Authors(request SeriesRequest) AuthorPage {
	query := strings.ToLower(strings.TrimSpace(request.Query))
	bucket := strings.ToUpper(strings.TrimSpace(request.Bucket))
	if query == "" && bucket == "" && request.LibraryID != "" {
		for _, library := range s.libraries {
			if request.LibraryID == library.library.ID {
				return library.authorPage(request)
			}
		}
		return AuthorPage{AnchorFound: request.AfterID == ""}
	}
	out := make([]AuthorResult, 0)
	for _, library := range s.libraries {
		if request.LibraryID != "" && request.LibraryID != library.library.ID {
			continue
		}
		for _, author := range library.authorSnapshot() {
			if query != "" && !strings.Contains(strings.ToLower(author.Name), query) {
				continue
			}
			if bucket != "" && author.Bucket != bucket {
				continue
			}
			out = append(out, author)
		}
	}
	sortAuthorResults(out)
	return paginateAuthors(out, request)
}

func paginateAuthors(out []AuthorResult, request SeriesRequest) AuthorPage {
	total := len(out)
	limit := request.Limit
	if limit <= 0 {
		limit = total
	} else if limit > MaxPageSize {
		limit = MaxPageSize
	}
	start := max(request.Offset, 0)
	anchorFound := request.AfterID == ""
	if request.AfterID != "" {
		start = 0
		for index, item := range out {
			if item.LibraryID == request.AfterLibraryID && item.Name == request.AfterID {
				start, anchorFound = index+1, true
				break
			}
		}
	}
	start = min(start, total)
	end := min(start+limit, total)
	if !anchorFound {
		end = start
	}
	return AuthorPage{Items: out[start:end], Total: total, AnchorFound: anchorFound, HasMore: anchorFound && end < total}
}

func sortAuthorResults(out []AuthorResult) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Unknown != out[j].Unknown {
			return !out[i].Unknown
		}
		left, right := catalog.SortKey(out[i].Name), catalog.SortKey(out[j].Name)
		if left != right {
			return catalog.NaturalKeyLess(left, right)
		}
		return out[i].LibraryID < out[j].LibraryID
	})
}

func (i *libraryIndex) seriesSnapshot() []SeriesResult {
	_ = i.snapshot()
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]SeriesResult(nil), i.series...)
}

func (i *libraryIndex) seriesPage(request SeriesRequest) SeriesPage {
	_ = i.snapshot()
	i.mu.RLock()
	defer i.mu.RUnlock()
	start := max(request.Offset, 0)
	anchorFound := request.AfterID == ""
	if request.AfterID != "" && request.AfterLibraryID == i.library.ID {
		if position, ok := i.seriesPositions[request.AfterID]; ok {
			start, anchorFound = position+1, true
		}
	}
	start = min(start, len(i.series))
	limit := request.Limit
	if limit <= 0 {
		limit = len(i.series)
	} else if limit > MaxPageSize {
		limit = MaxPageSize
	}
	end := min(start+limit, len(i.series))
	if !anchorFound {
		end = start
	}
	items := append([]SeriesResult(nil), i.series[start:end]...)
	return SeriesPage{Items: items, Total: len(i.series), AnchorFound: anchorFound, HasMore: anchorFound && end < len(i.series)}
}

func (i *libraryIndex) authorSnapshot() []AuthorResult {
	_ = i.snapshot()
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]AuthorResult(nil), i.authors...)
}

func (i *libraryIndex) authorPage(request SeriesRequest) AuthorPage {
	_ = i.snapshot()
	i.mu.RLock()
	defer i.mu.RUnlock()
	start := max(request.Offset, 0)
	anchorFound := request.AfterID == ""
	if request.AfterID != "" && request.AfterLibraryID == i.library.ID {
		if position, ok := i.authorPositions[request.AfterID]; ok {
			start, anchorFound = position+1, true
		}
	}
	start = min(start, len(i.authors))
	limit := request.Limit
	if limit <= 0 {
		limit = len(i.authors)
	} else if limit > MaxPageSize {
		limit = MaxPageSize
	}
	end := min(start+limit, len(i.authors))
	if !anchorFound {
		end = start
	}
	items := append([]AuthorResult(nil), i.authors[start:end]...)
	return AuthorPage{Items: items, Total: len(i.authors), AnchorFound: anchorFound, HasMore: anchorFound && end < len(i.authors)}
}

func buildSeriesIndex(library Library, books []indexedBook) []SeriesResult {
	counts := make(map[string]int)
	representatives := make(map[string]indexedBook)
	standaloneCount := 0
	var standalone indexedBook
	for _, indexed := range books {
		book := indexed.book
		if book.Series == "" {
			standaloneCount++
			if standalone.book.ID == "" || catalog.NaturalKeyLess(indexed.titleKey, standalone.titleKey) || indexed.titleKey == standalone.titleKey && book.ID < standalone.book.ID {
				standalone = indexed
			}
			continue
		}
		counts[book.Series]++
		current, exists := representatives[book.Series]
		if !exists || indexedSeriesLess(indexed, current) || !indexedSeriesLess(current, indexed) && book.ID < current.book.ID {
			representatives[book.Series] = indexed
		}
	}
	groups := make([]SeriesResult, 0, len(counts)+1)
	for name, count := range counts {
		groups = append(groups, SeriesResult{LibraryID: library.ID, LibraryName: library.Name, Name: name, Bucket: catalog.Bucket(name), Count: count, Representative: representatives[name].book})
	}
	if standaloneCount > 0 {
		groups = append(groups, SeriesResult{LibraryID: library.ID, LibraryName: library.Name, Name: "Standalone books", Count: standaloneCount, Unsorted: true, Representative: standalone.book})
	}
	sortSeriesResults(groups)
	return groups
}

func buildAuthorIndex(library Library, books []indexedBook) []AuthorResult {
	counts := make(map[string]int)
	representatives := make(map[string]indexedBook)
	displayNames := make(map[string]string)
	for _, indexed := range books {
		authors := indexed.book.Authors
		if len(authors) == 0 {
			authors = []string{"Unknown author"}
		}
		seen := make(map[string]bool, len(authors))
		for _, raw := range authors {
			name := strings.TrimSpace(raw)
			if name == "" {
				name = "Unknown author"
			}
			key := strings.ToLower(name)
			if seen[key] {
				continue
			}
			seen[key] = true
			counts[key]++
			if _, exists := displayNames[key]; !exists {
				displayNames[key] = name
			}
			if current, ok := representatives[key]; !ok || catalog.NaturalKeyLess(indexed.titleKey, current.titleKey) || indexed.titleKey == current.titleKey && indexed.book.ID < current.book.ID {
				representatives[key] = indexed
			}
		}
	}
	out := make([]AuthorResult, 0, len(counts))
	for key, count := range counts {
		name := displayNames[key]
		out = append(out, AuthorResult{LibraryID: library.ID, LibraryName: library.Name, Name: name, Bucket: catalog.Bucket(name), Count: count, Unknown: key == "unknown author", Representative: representatives[key].book})
	}
	sortAuthorResults(out)
	return out
}

func indexedSeriesLess(a, b indexedBook) bool {
	if a.book.Series == b.book.Series && a.pathDepth != b.pathDepth {
		return a.pathDepth < b.pathDepth
	}
	if a.book.SeriesIndex == nil && b.book.SeriesIndex != nil {
		return false
	}
	if a.book.SeriesIndex != nil && b.book.SeriesIndex == nil {
		return true
	}
	if a.book.SeriesIndex != nil && b.book.SeriesIndex != nil && *a.book.SeriesIndex != *b.book.SeriesIndex {
		return *a.book.SeriesIndex < *b.book.SeriesIndex
	}
	return catalog.NaturalKeyLess(a.titleKey, b.titleKey)
}
