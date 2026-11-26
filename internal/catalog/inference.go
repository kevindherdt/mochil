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
		// Every book counts towards its folder, parsed or not. Counting only the parsed ones meant that tightening the
		// bare-number rule (see bareNumberIsAPosition) also quietly stopped "86--EIGHTY-SIX/" from being a series
		// folder at all — the series vanished along with the bad title split.
		counts[filepath.ToSlash(filepath.Dir(filepath.FromSlash(book.Path)))]++
	}
	for i := range books {
		book := &books[i]
		if book.PublicationRole == "" {
			book.PublicationRole = ClassifyPublication(book.Title, book.Path)
		}
		inferred := parsed[i]
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(book.Path)))
		if options.FolderLayout == "series" && dir != "." && book.Series == "" && folderNamesSeries(dir, book.Title) {
			book.Series = displayFilename(filepath.Base(filepath.FromSlash(dir)))
		}
		// A folder holding a single book whose name is the book's own name is a per-book folder, not a series:
		// sandbox layouts store every book in "Author/Title (id)/". Calling that a series gave 1984 a series
		// named "1984" and a position of Book 1.
		seriesFolder := dir != "." && (counts[dir] >= 2 || (options.FolderLayout == "series" && folderNamesSeries(dir, book.Title)))
		if seriesFolder && book.Series == "" {
			book.Series = displayFilename(filepath.Base(filepath.FromSlash(dir)))
		}
		if !inferred.ok {
			if options.FilenameAuthor {
				inferFilenameAuthor(book, strings.TrimSuffix(filepath.Base(filepath.FromSlash(book.Path)), filepath.Ext(book.Path)))
			}
			inferMarkedSeriesIndex(book)
			continue
		}
		if seriesFolder && dir != "." {
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

// folderNamesSeries reports whether a folder name is something other than the book's own title, which is the
// only case where the folder carries information the book does not already have.
func folderNamesSeries(dir, title string) bool {
	name := strings.ToLower(displayFilename(filepath.Base(filepath.FromSlash(dir))))
	clean := strings.ToLower(strings.TrimSpace(title))
	if name == "" || clean == "" {
		return true
	}
	if name == clean {
		return false
	}
	// "1984 (282)" is a per-book folder too: the id suffix is storage, not part of any series name.
	if open := strings.LastIndex(name, " ("); open > 0 && strings.HasSuffix(name, ")") {
		if strings.TrimSpace(name[:open]) == clean {
			return false
		}
	}
	return true
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
		// A bare leading number is the weakest evidence there is, and taking it on faith corrupts titles: "1984 - A
		// Novel" became volume 1984 titled "A Novel", and "86--EIGHTY-SIX - Volume 01" — a series whose own name starts
		// with a number — became volume 86 titled "-EIGHTY-SIX - Volume 01", which is the leading dash in the library.
		// So it only counts when what follows actually reads as a title: no leading separator (a rule that leaves one
		// has mis-split the name), something left after trimming, and a number small enough to be a position rather
		// than a year or a title.
		if pattern == bareNumberedFilename && !bareNumberIsAPosition(matches[1], matches[2], value) {
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

// bareNumberIsAPosition decides whether a number at the front of a filename is a volume or part of the name.
//
// The marked forms ("Volume 7", "#7", "Book 7") say what they mean; a bare `7 - Title` only suggests it, and the cost
// of being wrong is a mangled title in the catalogue that someone then has to find and fix by hand.
func bareNumberIsAPosition(number, remainder, whole string) bool {
	if len(number) >= 4 { // a year, or a title that is a number: 1984, 2001, 11.22
		return false
	}
	rest := strings.TrimLeft(remainder, " \t")
	if rest == "" || strings.ContainsAny(rest[:1], "-–—_") {
		return false // nothing left to call a title, or the split landed inside a separator
	}
	// "86--EIGHTY-SIX - Volume 01": the series itself begins with the number, so the number is part of the name.
	name := strings.TrimSuffix(whole, filepath.Ext(whole))
	if trimmed := strings.TrimLeft(displayFilename(name), " "); trimmed != "" {
		first := trimmed
		if fields := strings.Fields(trimmed); len(fields) > 0 {
			first = fields[0]
		}
		if strings.HasPrefix(strings.ReplaceAll(strings.ToLower(first), "--", "-"), strings.ToLower(number)+"-") {
			return false
		}
	}
	return true
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
