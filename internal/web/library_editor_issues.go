package web

import (
	"bytes"
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"mochil/internal/catalog"
)

// editorIssues names what a person should look at on each book, one entry per book in the order given. Duplicates
// need the whole library, so they come first. The result is indexed rather than keyed by book id: the caller builds
// a row from each book anyway, so the id map bought 5 400 inserts and 13 000 hashes of a value the loop already had.
func editorIssues(books []catalog.Book) [][]string {
	issues := make([][]string, len(books))
	markDuplicateVolumes(books, issues)
	for index, book := range books {
		if len(book.Authors) == 0 || strings.EqualFold(strings.TrimSpace(book.Authors[0]), "unknown") {
			issues[index] = append(issues[index], "no_author")
		}
		if book.Series != "" && book.SeriesIndex == nil {
			issues[index] = append(issues[index], "no_number")
		}
		if containsLowered(book.Title, "side stor") {
			issues[index] = append(issues[index], "side_story")
		}
	}
	return issues
}

// seriesVolume is one book's series and volume as the duplicate scan compares them: the series folded into a shared
// buffer, the volume as its bits, and the position of the book in the list.
type seriesVolume struct {
	series []byte
	volume uint64
	book   int32
}

// markDuplicateVolumes flags every book that shares a series and a volume with another book. Two passes over the
// library rather than a map of keys: the folded series names live in one buffer and the books are sorted by their
// key, so each run of equal keys is one group. The version this replaced built a "series\x00number" string per book
// (5 400 strings and 5 400 FormatFloat results) and a []int per series in a map — 10 800 allocations and 1.6 MB for
// one editor chunk, of which this scan is most.
func markDuplicateVolumes(books []catalog.Book, issues [][]string) {
	size := 0
	for i := range books {
		if books[i].Series != "" && books[i].SeriesIndex != nil {
			size += len(books[i].Series)
		}
	}
	arena := make([]byte, 0, size)
	entries := make([]seriesVolume, 0, len(books))
	for i := range books {
		book := &books[i]
		if book.Series == "" || book.SeriesIndex == nil {
			continue
		}
		var folded []byte
		arena, folded = appendFolded(arena, book.Series)
		entries = append(entries, seriesVolume{series: folded, volume: volumeBits(*book.SeriesIndex), book: int32(i)})
	}
	if len(entries) < 2 {
		return
	}
	slices.SortFunc(entries, func(a, b seriesVolume) int {
		if order := bytes.Compare(a.series, b.series); order != 0 {
			return order
		}
		return cmp.Compare(a.volume, b.volume)
	})
	for start := 0; start < len(entries); {
		end := start + 1
		for end < len(entries) && entries[end].volume == entries[start].volume && bytes.Equal(entries[end].series, entries[start].series) {
			end++
		}
		if end-start > 1 {
			for _, entry := range entries[start:end] {
				issues[entry.book] = append(issues[entry.book], "duplicate")
			}
		}
		start = end
	}
}

// volumeBits is a volume number as the duplicate scan compares it. Sorting by bits is a total order over the values,
// and two books are grouped exactly when the number string this used to build would have been the same: every other
// value formats on its own, so bit equality is that equality. The two cases where the two disagree are handled here —
// every NaN used to format as "NaN" and is folded to one value, and -0 stays distinct from 0 because it formatted as
// "-0" rather than "0".
func volumeBits(volume float64) uint64 {
	if math.IsNaN(volume) {
		return math.Float64bits(math.NaN())
	}
	return math.Float64bits(volume)
}

// containsLowered reports whether lowering s would leave word inside it, without building the lowered string.
// strings.ToLower allocates a string whenever the text has a capital in it, which for a book title is most of them,
// and one chunk of the editor asked it about every book in the library to look for one phrase. Text that is plain
// ASCII is compared a byte at a time, which is what strings.ToLower does to it; anything else takes that path so the
// answer stays exactly what it was.
func containsLowered(s, word string) bool {
	if word == "" {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return strings.Contains(strings.ToLower(s), word)
		}
	}
	for start := 0; start+len(word) <= len(s); start++ {
		if !equalFoldASCII(s[start:start+len(word)], word) {
			continue
		}
		return true
	}
	return false
}

// equalFoldASCII compares two plain ASCII strings a byte at a time, folding Latin capitals the way strings.ToLower
// folds them.
func equalFoldASCII(s, word string) bool {
	for i := 0; i < len(s); i++ {
		left, right := s[i], word[i]
		if 'A' <= left && left <= 'Z' {
			left += 'a' - 'A'
		}
		if 'A' <= right && right <= 'Z' {
			right += 'a' - 'A'
		}
		if left != right {
			return false
		}
	}
	return true
}
