package web

import (
	"strings"
	"time"

	"mochil/internal/catalog"
	catalogquery "mochil/internal/query"
	"mochil/internal/reading"
)

// Reading states a smart collection can ask for. They are the person's own: someone else's progress never counts.
const (
	statusUnread   = "unread"
	statusReading  = "reading"
	statusFinished = "finished"
)

const maximumAddedDays = 3650

// normalized trims the text fields and drops values that mean nothing, so the same rule is always stored the same way.
func (r collectionRule) normalized() collectionRule {
	r.Query, r.Author, r.Series = strings.TrimSpace(r.Query), strings.TrimSpace(r.Author), strings.TrimSpace(r.Series)
	r.Format, r.Tag = strings.ToLower(strings.TrimSpace(r.Format)), strings.TrimSpace(r.Tag)
	r.Status = strings.ToLower(strings.TrimSpace(r.Status))
	if r.AddedDays < 0 {
		r.AddedDays = 0
	}
	return r
}

// valid says whether a rule can be saved: something to filter on, nothing absurd, and a known reading state.
func (r collectionRule) valid() bool {
	for _, text := range []string{r.Query, r.Author, r.Series, r.Format, r.Tag} {
		if len(text) > 200 {
			return false
		}
	}
	switch r.Status {
	case "", statusUnread, statusReading, statusFinished:
	default:
		return false
	}
	return !r.empty() && r.AddedDays <= maximumAddedDays
}

// needsReading is true when matching depends on the person's reading places, so they only get looked up then.
func (r collectionRule) needsReading() bool { return r.Status != "" }

// matchesDetails applies the parts of a rule the catalogue search cannot: reading state, format, tag and age.
func (r collectionRule) matchesDetails(book catalog.Book, position reading.Position, hasPosition bool, now time.Time) bool {
	if r.Format != "" && !strings.EqualFold(book.Format, r.Format) {
		return false
	}
	if r.Tag != "" {
		found := false
		for _, tag := range book.Tags {
			if strings.EqualFold(tag, r.Tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if r.AddedDays > 0 && book.Modified.Before(now.AddDate(0, 0, -r.AddedDays)) {
		return false
	}
	switch r.Status {
	case statusUnread:
		return !hasPosition || (!position.Finished && position.Fraction <= 0)
	case statusReading:
		return hasPosition && !position.Finished && position.Fraction > 0
	case statusFinished:
		return hasPosition && position.Finished
	}
	return true
}

// maximumVolume is the biggest volume number that still counts as a volume: a series numbered by year (2019, 2020…)
// would otherwise look as if thousands of volumes were missing.
const maximumVolume = 300

// extrasFrom is where a fractional series number stops being a split volume and starts being an extra.
const extrasFrom = 90

func gapKey(libraryID, series string) string { return libraryID + "\x00" + strings.ToLower(series) }

// gapSeries says which series skip a volume: the ones with at least two numbered volumes where
// something between volume 1 and the highest one is absent. Side stories and other extras never count as volumes.
func gapSeries(books []catalogItem) map[string]bool {
	have := map[string]map[int]bool{}
	for _, item := range books {
		book := item.Book
		if book.Series == "" || book.SeriesIndex == nil || book.PublicationRole != "" {
			continue
		}
		// A split volume (4.1, 4.2) counts as its whole number: the volume is there, in parts.
		whole := int(*book.SeriesIndex)
		// Publishers number extras 99.9 or the like; a lone one must not make volumes 13 to 98 look missing.
		if *book.SeriesIndex >= extrasFrom && *book.SeriesIndex != float64(whole) {
			continue
		}
		index := whole
		if index < 1 || index > maximumVolume {
			continue
		}
		key := gapKey(item.LibraryID, book.Series)
		if have[key] == nil {
			have[key] = map[int]bool{}
		}
		have[key][index] = true
	}
	gaps := map[string]bool{}
	for key, volumes := range have {
		highest := 0
		for volume := range volumes {
			highest = max(highest, volume)
		}
		if len(volumes) >= 2 && len(volumes) < highest {
			gaps[key] = true
		}
	}
	return gaps
}

// seriesWithGaps looks at the whole library, not at the rest of the rule, so a filter never hides a volume that is there.
func (m *multiServer) seriesWithGaps(libraryID string) map[string]bool {
	request := catalogquery.BooksRequest{LibraryID: libraryID, Limit: catalogquery.MaxPageSize}
	var all []catalogItem
	for {
		page := m.query.Books(request)
		for _, item := range page.Items {
			all = append(all, catalogItem{LibraryID: item.LibraryID, Book: item.Book})
		}
		if !page.HasMore || len(page.Items) == 0 {
			return gapSeries(all)
		}
		request.Offset += len(page.Items)
	}
}

type catalogItem struct {
	LibraryID string
	Book      catalog.Book
}
