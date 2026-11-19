package web

import (
	"strconv"
	"strings"
	"unicode"

	"mochil/internal/catalog"
	"mochil/internal/ingest"
)

// Import statuses: what an incoming book is compared with in the library it would join.
const (
	importInLibrary = "in_library" // the same series and number (or title and author) is already there
	importNewVolume = "new_volume" // the series is in the library but this volume is not
	importPossible  = "possible"   // a book with the same title is there, but nothing else confirms it
	importNew       = "new"        // nothing like it is in the library
)

type importExisting struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Series string `json:"series,omitempty"`
}

type importRow struct {
	ingest.DiscoveryItem
	Status   string          `json:"status"`
	Copied   bool            `json:"copied,omitempty"`
	Existing *importExisting `json:"existing,omitempty"`
}

// importKey lowercases and drops everything but letters and digits, and a trailing "(2018)" style year, so "Naruto
// Shinden (2018)" and "naruto shinden" compare equal.
func importKey(value string) string {
	value = strings.TrimSpace(value)
	if open := strings.LastIndex(value, "("); open > 0 && strings.HasSuffix(value, ")") {
		if year, err := strconv.Atoi(strings.TrimSpace(value[open+1 : len(value)-1])); err == nil && year > 1000 && year < 3000 {
			value = value[:open]
		}
	}
	var out strings.Builder
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			out.WriteRune(character)
		}
	}
	return out.String()
}

func importIndex(index *float64) string {
	if index == nil {
		return ""
	}
	return strconv.FormatFloat(*index, 'f', -1, 64)
}

// matchImports compares discovered books with the library they would join. It reads both sides only.
func matchImports(items []ingest.DiscoveryItem, books []catalog.Book) []importRow {
	bySeriesNumber := map[string]catalog.Book{}
	seriesKnown := map[string]bool{}
	byTitleAuthor := map[string]catalog.Book{}
	byTitle := map[string]catalog.Book{}
	for _, book := range books {
		series := importKey(book.Series)
		if series != "" {
			seriesKnown[series] = true
			if index := importIndex(book.SeriesIndex); index != "" {
				if _, taken := bySeriesNumber[series+"|"+index]; !taken {
					bySeriesNumber[series+"|"+index] = book
				}
			}
		}
		title := importKey(book.Title)
		if title == "" {
			continue
		}
		if len(book.Authors) > 0 {
			byTitleAuthor[title+"|"+importKey(book.Authors[0])] = book
		}
		if _, taken := byTitle[title]; !taken {
			byTitle[title] = book
		}
	}
	rows := make([]importRow, len(items))
	for i, item := range items {
		row := importRow{DiscoveryItem: item, Status: importNew}
		proposal := item.Proposal
		series := importKey(proposal.Series)
		var found *catalog.Book
		if series != "" {
			if index := importIndex(proposal.SeriesIndex); index != "" {
				if book, ok := bySeriesNumber[series+"|"+index]; ok {
					found = &book
					row.Status = importInLibrary
				}
			}
			if found == nil && seriesKnown[series] {
				row.Status = importNewVolume
			}
		}
		if found == nil {
			title := importKey(proposal.Title)
			if len(proposal.Authors) > 0 && title != "" {
				if book, ok := byTitleAuthor[title+"|"+importKey(proposal.Authors[0])]; ok {
					found, row.Status = &book, importInLibrary
				}
			}
			// a same-titled book is a hint only, and is skipped when the volume number is known and simply not in the library
			if found == nil && title != "" && (row.Status != importNewVolume || importIndex(proposal.SeriesIndex) == "") {
				if book, ok := byTitle[title]; ok {
					found, row.Status = &book, importPossible
				}
			}
		}
		if found != nil {
			row.Existing = &importExisting{ID: found.ID, Title: found.Title, Series: found.Series}
		}
		rows[i] = row
	}
	return rows
}
