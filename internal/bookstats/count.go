// Package bookstats counts the words and pages of a book from its file: a measurement of what is in the book, kept
// beside the catalogue and never written into it.
package bookstats

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strings"
	"sync"
	"unicode"
)

// WordsPerPage turns words into an estimated page count, the way Calibre's page-count plugins do for a book that has
// no fixed pages: it is a guide for comparing books, not a printed page number.
const WordsPerPage = 250

// ErrUnsupported: this format has nothing to count (or no way to count it yet).
var ErrUnsupported = errors.New("this format cannot be counted")

type Stats struct {
	Words int `json:"words"`
	Pages int `json:"pages"`
}

// ReadSeeker is what a catalogue hands out for a book's file.
type ReadSeeker interface {
	io.Reader
	io.Seeker
}

// Count measures one book. An EPUB is counted from the text of its reading order; a comic archive already knows its
// pages (pageCount), and has no words.
func Count(file ReadSeeker, format string, pageCount int) (Stats, error) {
	switch strings.ToLower(format) {
	case "epub":
		return countEPUB(file)
	case "cbz":
		if pageCount > 0 {
			return Stats{Pages: pageCount}, nil
		}
	}
	return Stats{}, ErrUnsupported
}

// seekReaderAt lets a seekable file be opened as a zip without reading it all into memory.
type seekReaderAt struct {
	mu   sync.Mutex
	file ReadSeeker
}

func (s *seekReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.file.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(s.file, p)
}

func countEPUB(file ReadSeeker) (Stats, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return Stats{}, err
	}
	archive, err := zip.NewReader(&seekReaderAt{file: file}, size)
	if err != nil {
		return Stats{}, err
	}
	files := map[string]*zip.File{}
	for _, entry := range archive.File {
		files[entry.Name] = entry
	}
	var container struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := decode(files["META-INF/container.xml"], &container); err != nil || len(container.Rootfiles) == 0 {
		return Stats{}, errors.New("not a readable EPUB")
	}
	opfPath := container.Rootfiles[0].FullPath
	var pkg struct {
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef  string `xml:"idref,attr"`
			Linear string `xml:"linear,attr"`
		} `xml:"spine>itemref"`
	}
	if err := decode(files[opfPath], &pkg); err != nil {
		return Stats{}, errors.New("not a readable EPUB")
	}
	hrefs := map[string]string{}
	for _, item := range pkg.Manifest {
		hrefs[item.ID] = item.Href
	}
	words := 0
	for _, ref := range pkg.Spine {
		href, ok := hrefs[ref.IDRef]
		if !ok {
			continue
		}
		name := path.Join(path.Dir(opfPath), unescapeHref(href))
		entry := files[name]
		if entry == nil {
			continue
		}
		n, err := countWords(entry)
		if err != nil {
			continue // one broken chapter should not hide the rest of the book
		}
		words += n
	}
	return Stats{Words: words, Pages: pagesFor(words)}, nil
}

func pagesFor(words int) int {
	if words <= 0 {
		return 0
	}
	return (words + WordsPerPage - 1) / WordsPerPage
}

func unescapeHref(href string) string {
	if i := strings.IndexAny(href, "#?"); i >= 0 {
		href = href[:i]
	}
	replacer := strings.NewReplacer("%20", " ", "%28", "(", "%29", ")", "%27", "'", "%2C", ",", "%26", "&")
	return replacer.Replace(href)
}

func decode(entry *zip.File, into any) error {
	if entry == nil {
		return errors.New("missing file")
	}
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return xml.NewDecoder(io.LimitReader(reader, 16<<20)).Decode(into)
}

// countWords counts the words a reader would see in one chapter: text outside <head>, <script> and <style>.
func countWords(entry *zip.File) (int, error) {
	reader, err := entry.Open()
	if err != nil {
		return 0, err
	}
	defer reader.Close()
	decoder := xml.NewDecoder(io.LimitReader(reader, 64<<20))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	skip, words := 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return words, nil
		}
		if err != nil {
			return words, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToLower(t.Name.Local) {
			case "head", "script", "style":
				skip++
			}
		case xml.EndElement:
			switch strings.ToLower(t.Name.Local) {
			case "head", "script", "style":
				if skip > 0 {
					skip--
				}
			}
		case xml.CharData:
			if skip == 0 {
				for _, field := range strings.Fields(string(t)) {
					if strings.IndexFunc(field, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
						words++
					}
				}
			}
		}
	}
}
