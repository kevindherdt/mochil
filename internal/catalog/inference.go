package catalog

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	markedNumberedFilename  = regexp.MustCompile(`(?i)^(?:v(?:ol(?:ume)?)?\.?|#)\s*(\d+(?:\.\d+)?)\s*(?:[-_]\s*|\s+)(.+)$`)
	bracketNumberedFilename = regexp.MustCompile(`(?i)^[\[(]\s*(\d+(?:\.\d+)?)\s*[\])]\s*(?:[-_]\s*)?(.+)$`)
	bareNumberedFilename    = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*[-_]\s*(.+)$`)
)
var markedVolume = regexp.MustCompile(`(?i)(?:^|\s)(?:v(?:ol(?:ume)?)?\.?|book|#)\s*(\d+(?:\.\d+)?)(?:\s|$)`)

type inferredName struct {
	index float64
	title string
	ok    bool
}

func inferFilesystemMetadata(books []Book, options FilesystemOptions) {
	if options.FolderLayout == "metadata-only" {
		return
	}
	parsed := make([]inferredName, len(books))
	counts := make(map[string]int)
	for i, book := range books {
		base := strings.TrimSuffix(filepath.Base(filepath.FromSlash(book.Path)), filepath.Ext(book.Path))
		parsed[i] = parseNumberedFilename(base)
		if parsed[i].ok {
			counts[filepath.ToSlash(filepath.Dir(filepath.FromSlash(book.Path)))]++
		}
	}
	for i := range books {
		book := &books[i]
		if book.PublicationRole == "" {
			book.PublicationRole = ClassifyPublication(book.Title, book.Path)
		}
		inferred := parsed[i]
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(book.Path)))
		if options.FolderLayout == "series" && dir != "." && book.Series == "" {
			book.Series = displayFilename(filepath.Base(filepath.FromSlash(dir)))
		}
		if !inferred.ok {
			if options.FilenameAuthor {
				inferFilenameAuthor(book, strings.TrimSuffix(filepath.Base(filepath.FromSlash(book.Path)), filepath.Ext(book.Path)))
			}
			inferMarkedSeriesIndex(book)
			continue
		}
		seriesFolder := options.FolderLayout == "series" || (options.FolderLayout == "auto" && counts[dir] >= 2)
		if seriesFolder && dir != "." {
			if book.Series == "" {
				book.Series = displayFilename(filepath.Base(filepath.FromSlash(dir)))
			}
			if book.SeriesIndex == nil && book.PublicationRole == "" {
				index := inferred.index
				if marked := markedSeriesIndex(inferred.title); marked != nil {
					index = *marked
				}
				book.SeriesIndex = &index
			}
		}
		base := strings.TrimSuffix(filepath.Base(filepath.FromSlash(book.Path)), filepath.Ext(book.Path))
		if book.Title == base {
			book.Title = inferred.title
		}
		if options.FilenameAuthor {
			inferFilenameAuthor(book, inferred.title)
		}
		inferMarkedSeriesIndex(book)
	}
}

func inferMarkedSeriesIndex(book *Book) {
	if book.Series == "" || book.SeriesIndex != nil || book.PublicationRole != "" {
		return
	}
	book.SeriesIndex = markedSeriesIndex(book.Title)
}

func markedSeriesIndex(value string) *float64 {
	matches := markedVolume.FindStringSubmatch(value)
	if len(matches) != 2 {
		return nil
	}
	if index, err := strconv.ParseFloat(matches[1], 64); err == nil {
		return &index
	}
	return nil
}

func parseNumberedFilename(value string) inferredName {
	value = strings.TrimSpace(value)
	for _, pattern := range []*regexp.Regexp{markedNumberedFilename, bracketNumberedFilename, bareNumberedFilename} {
		matches := pattern.FindStringSubmatch(value)
		if len(matches) != 3 {
			continue
		}
		index, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return inferredName{}
		}
		return inferredName{index: index, title: displayFilename(matches[2]), ok: true}
	}
	return inferredName{}
}

func displayFilename(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "_", " ")), " ")
}

func inferFilenameAuthor(book *Book, value string) {
	if len(book.Authors) != 1 || book.Authors[0] != "Unknown" {
		return
	}
	separator := strings.LastIndex(value, " - ")
	if separator <= 0 || separator+3 >= len(value) {
		return
	}
	title := strings.TrimSpace(value[:separator])
	author := strings.TrimSpace(value[separator+3:])
	if title == "" || author == "" {
		return
	}
	book.Title = title
	book.Authors = []string{author}
	book.AuthorSort = []string{author}
}
