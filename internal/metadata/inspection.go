// Package metadata contains storage-independent metadata review rules.
package metadata

import (
	"sort"
	"strings"

	"mochil/internal/catalog"
)

type Issue struct {
	BookID      string   `json:"book_id"`
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Authors     []string `json:"authors,omitempty"`
	Series      string   `json:"series,omitempty"`
	SeriesIndex *float64 `json:"series_index,omitempty"`
	Format      string   `json:"format,omitempty"`
	Code        string   `json:"code"`
	Reason      string   `json:"reason"`
	Confidence  string   `json:"confidence"`
}

type Report struct {
	BooksInspected int            `json:"books_inspected"`
	IssueCount     int            `json:"issue_count"`
	Counts         map[string]int `json:"counts"`
	Issues         []Issue        `json:"issues"`
}

// Inspect evaluates catalogue metadata without opening or modifying publications.
func Inspect(books []catalog.Book) Report {
	report := Report{BooksInspected: len(books), Counts: make(map[string]int)}
	seriesIndexes := make(map[string]map[float64][]catalog.Book)
	editions := make(map[string][]catalog.Book)
	for _, book := range books {
		title := strings.TrimSpace(book.Title)
		if title == "" {
			report.add(book, "missing_title", "No title was found.", "high")
		}
		if len(book.Authors) == 0 || allBlank(book.Authors) || allUnknown(book.Authors) {
			reason := "No usable author was found."
			if strings.EqualFold(strings.TrimSpace(book.Format), "pdf") {
				reason = "This PDF does not provide a usable author. Enter one manually, or keep it as Unknown."
			}
			report.add(book, "missing_author", reason, "high")
		}
		if strings.TrimSpace(book.Series) != "" && book.SeriesIndex == nil && !specialPublication(book) {
			if _, actionable := titleVolumeNumber(book.Title); actionable {
				report.add(book, "missing_series_index", "The title contains a volume number, but the catalogue does not.", "medium")
			}
		}
		if book.SeriesIndex != nil && strings.TrimSpace(book.Series) != "" {
			key := strings.ToLower(strings.TrimSpace(book.Series))
			if seriesIndexes[key] == nil {
				seriesIndexes[key] = make(map[float64][]catalog.Book)
			}
			seriesIndexes[key][*book.SeriesIndex] = append(seriesIndexes[key][*book.SeriesIndex], book)
		}
		if key := editionKey(book); key != "" {
			editions[key] = append(editions[key], book)
		}
	}
	for _, duplicates := range editions {
		if len(duplicates) < 2 {
			continue
		}
		for _, book := range duplicates {
			report.add(book, "duplicate_edition", "Another publication on this shelf has the same title, series, and author. Keep both if they are intentional editions.", "medium")
		}
	}
	for _, indexes := range seriesIndexes {
		for index, duplicates := range indexes {
			orderCandidates := make([]catalog.Book, 0, len(duplicates))
			for _, book := range duplicates {
				if !specialPublication(book) {
					orderCandidates = append(orderCandidates, book)
				}
			}
			if len(orderCandidates) < 2 {
				continue
			}
			if duplicateVolumeIsIntentional(orderCandidates, index) {
				continue
			}
			for _, book := range orderCandidates {
				report.add(book, "duplicate_series_index", "Another book in this series has the same volume number.", "high")
			}
		}
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Code != report.Issues[j].Code {
			return report.Issues[i].Code < report.Issues[j].Code
		}
		if report.Issues[i].Title != report.Issues[j].Title {
			return report.Issues[i].Title < report.Issues[j].Title
		}
		return report.Issues[i].BookID < report.Issues[j].BookID
	})
	report.IssueCount = len(report.Issues)
	return report
}

func editionKey(book catalog.Book) string {
	title := normalizedEditionTitle(book.Title)
	if title == "" {
		return ""
	}
	authors := make([]string, 0, len(book.Authors))
	for _, author := range book.Authors {
		value := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(author)), " "))
		if value != "" && value != "unknown" && value != "unknown author" {
			authors = append(authors, value)
		}
	}
	return strings.ToLower(strings.TrimSpace(book.Series)) + "\x00" + title + "\x00" + strings.Join(authors, "&")
}

func specialPublication(book catalog.Book) bool {
	return book.PublicationRole != "" || catalog.ClassifyPublication(book.Title, book.Path) != ""
}

func duplicateVolumeIsIntentional(books []catalog.Book, index float64) bool {
	allExplicit := true
	allSameTitle := true
	firstTitle := normalizedEditionTitle(books[0].Title)
	for _, book := range books {
		marker, explicit := titleVolumeNumber(book.Title)
		if explicit && marker != index {
			// At least one title directly contradicts the stored number. This is
			// actionable even when another edition legitimately uses the index.
			return false
		}
		if !explicit {
			allExplicit = false
		}
		if normalizedEditionTitle(book.Title) != firstTitle {
			allSameTitle = false
		}
	}
	// Parallel editions/subseries explicitly carrying the same number are not
	// a volume-order error. Exact duplicate titles are a separate dedup concern.
	return allExplicit || allSameTitle
}

func normalizedEditionTitle(title string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(title))), " ")
}

func (r *Report) add(book catalog.Book, code, reason, confidence string) {
	r.Issues = append(r.Issues, Issue{BookID: book.ID, Path: book.Path, Title: book.Title, Authors: book.Authors, Series: book.Series, SeriesIndex: book.SeriesIndex, Format: book.Format, Code: code, Reason: reason, Confidence: confidence})
	r.Counts[code]++
}

func allBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func allUnknown(values []string) bool {
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized != "" && normalized != "unknown" && normalized != "unknown author" {
			return false
		}
	}
	return true
}
