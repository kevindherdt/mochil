package catalog

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"
)

const maximumPageBytes = 128 << 20

// maximumCachedPageLists bounds how many archives keep their page order. One
// page request is enough to pay for the sort, so caching the few comics a
// reader currently has open takes the cost off every page after the first.
const maximumCachedPageLists = 16

// utf8RuneSelf is the first byte value that can begin a non-ASCII rune.
const utf8RuneSelf = 0x80

// pageCacheKey identifies one version of an archive. A rewritten file changes
// its size or modification time, which makes a cached order unusable rather
// than silently wrong.
type pageCacheKey struct {
	path     string
	size     int64
	modified int64
}

// pageEntry is one page image's archive position, with its ASCII-ness decided
// once so the comparator never rescans a name.
type pageEntry struct {
	name  string
	index int
	ascii bool
}

type zipPageReader struct {
	page io.ReadCloser
	zip  *zip.ReadCloser
}

func (r *zipPageReader) Read(buffer []byte) (int, error) { return r.page.Read(buffer) }
func (r *zipPageReader) Close() error {
	pageErr := r.page.Close()
	zipErr := r.zip.Close()
	if pageErr != nil {
		return pageErr
	}
	return zipErr
}

func (s *Store) Page(_ context.Context, book Book, index int) (Page, error) {
	if book.Format != "cbz" || index < 0 {
		return Page{}, errors.New("page streaming is not available")
	}
	path, err := s.safeAbsolutePath(book)
	if err != nil {
		return Page{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Page{}, err
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return Page{}, err
	}
	pages := s.pageOrder(path, info, zr)
	if index >= len(pages) || pages[index] >= len(zr.File) {
		_ = zr.Close()
		return Page{}, errors.New("page index is out of range")
	}
	entry := zr.File[pages[index]]
	if entry.UncompressedSize64 > maximumPageBytes {
		_ = zr.Close()
		return Page{}, errors.New("page exceeds the decompressed size limit")
	}
	reader, err := entry.Open()
	if err != nil {
		_ = zr.Close()
		return Page{}, err
	}
	return Page{
		Reader:    &zipPageReader{page: reader, zip: zr},
		MediaType: pageMediaType(entry.Name),
		Size:      int64(entry.UncompressedSize64),
	}, nil
}

// pageOrder returns the archive indexes of the page images in reading order,
// sorting at most once per (path, size, mtime). Streaming a 200-page comic used
// to re-sort the whole index, rune by rune, for every page.
//
// The cache stores positions, not *zip.File values: those belong to one open
// archive and cannot outlive its Close.
func (s *Store) pageOrder(path string, info os.FileInfo, zr *zip.ReadCloser) []int {
	key := pageCacheKey{path: path, size: info.Size(), modified: info.ModTime().UnixNano()}
	s.pagesMu.Lock()
	cached, ok := s.pageLists[key]
	s.pagesMu.Unlock()
	if ok && len(cached) <= len(zr.File) {
		return cached
	}
	entries := make([]pageEntry, 0, len(zr.File))
	for index, entry := range zr.File {
		if isPageImage(entry.Name) && !entry.FileInfo().IsDir() {
			entries = append(entries, pageEntry{name: entry.Name, index: index, ascii: isASCII(entry.Name)})
		}
	}
	sort.SliceStable(entries, func(left, right int) bool { return pageLess(entries[left], entries[right]) })
	order := make([]int, len(entries))
	for position, entry := range entries {
		order[position] = entry.index
	}
	s.pagesMu.Lock()
	if s.pageLists == nil {
		s.pageLists = map[pageCacheKey][]int{}
	}
	if previous, exists := s.pageLists[key]; exists && len(previous) <= len(zr.File) {
		s.pagesMu.Unlock()
		return previous // a concurrent request finished the same list first
	}
	for len(s.pageQueue) >= maximumCachedPageLists {
		delete(s.pageLists, s.pageQueue[0])
		s.pageQueue = s.pageQueue[1:]
	}
	s.pageLists[key] = order
	s.pageQueue = append(s.pageQueue, key)
	s.pagesMu.Unlock()
	return order
}

// pageLess keeps the reading order a person expects -- page2 before page10 --
// with a byte comparison for the ASCII names nearly every archive uses.
func pageLess(left, right pageEntry) bool {
	if left.ascii && right.ascii {
		return naturalLessASCII(left.name, right.name)
	}
	return naturalLessRunes(left.name, right.name)
}

func isASCII(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] >= utf8RuneSelf {
			return false
		}
	}
	return true
}

func isASCIIDigit(value byte) bool { return value >= '0' && value <= '9' }

// lowerASCII lowercases one ASCII byte; non-ASCII names never reach it.
func lowerASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}

// naturalLessASCII is naturalLess without the two []rune conversions that made
// every comparator call allocate. It is only used when both names are ASCII,
// where byte order and rune order agree.
func naturalLessASCII(left, right string) bool {
	li, ri := 0, 0
	for li < len(left) && ri < len(right) {
		a, b := lowerASCII(left[li]), lowerASCII(right[ri])
		if isASCIIDigit(a) && isASCIIDigit(b) {
			aStart, bStart := li, ri
			for li < len(left) && isASCIIDigit(left[li]) {
				li++
			}
			for ri < len(right) && isASCIIDigit(right[ri]) {
				ri++
			}
			aNumber := strings.TrimLeft(left[aStart:li], "0")
			bNumber := strings.TrimLeft(right[bStart:ri], "0")
			if aNumber == "" {
				aNumber = "0"
			}
			if bNumber == "" {
				bNumber = "0"
			}
			if len(aNumber) != len(bNumber) {
				return len(aNumber) < len(bNumber)
			}
			if aNumber != bNumber {
				return aNumber < bNumber
			}
			continue
		}
		if a != b {
			return a < b
		}
		li++
		ri++
	}
	// The prefixes compared equal, so the remaining lengths decide.
	return len(left)-li < len(right)-ri
}

func naturalLess(left, right string) bool {
	if isASCII(left) && isASCII(right) {
		return naturalLessASCII(left, right)
	}
	return naturalLessRunes(left, right)
}

func naturalLessRunes(left, right string) bool {
	a := []rune(strings.ToLower(left))
	b := []rune(strings.ToLower(right))
	for ai, bi := 0, 0; ai < len(a) && bi < len(b); {
		if unicode.IsDigit(a[ai]) && unicode.IsDigit(b[bi]) {
			aStart, bStart := ai, bi
			for ai < len(a) && unicode.IsDigit(a[ai]) {
				ai++
			}
			for bi < len(b) && unicode.IsDigit(b[bi]) {
				bi++
			}
			aNumber := strings.TrimLeft(string(a[aStart:ai]), "0")
			bNumber := strings.TrimLeft(string(b[bStart:bi]), "0")
			if aNumber == "" {
				aNumber = "0"
			}
			if bNumber == "" {
				bNumber = "0"
			}
			if len(aNumber) != len(bNumber) {
				return len(aNumber) < len(bNumber)
			}
			if aNumber != bNumber {
				return aNumber < bNumber
			}
			continue
		}
		if a[ai] != b[bi] {
			return a[ai] < b[bi]
		}
		ai++
		bi++
	}
	return len(a) < len(b)
}
