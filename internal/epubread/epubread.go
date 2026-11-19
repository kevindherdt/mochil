// Package epubread opens an EPUB far enough to look at what is in it: its identifiers, its reading order and the text
// of each chapter. It only reads, and it never loads a whole book into memory.
package epubread

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
	"sync"
)

// ReadSeeker is what a catalogue hands out for a book's file.
type ReadSeeker interface {
	io.Reader
	io.Seeker
}

type Identifier struct{ Scheme, Value string }

// Document is one file of the reading order.
type Document struct {
	Name  string // path inside the archive
	Guide string // what the book's own guide calls it ("copyright-page", "title-page"...), if it does
}

type Package struct {
	Identifiers []Identifier
	Spine       []Document
	files       map[string]*zip.File
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

// ZIP opens any zip archive (an EPUB or a comic) from a seekable file.
func ZIP(file ReadSeeker) (*zip.Reader, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	return zip.NewReader(&seekReaderAt{file: file}, size)
}

func Open(file ReadSeeker) (*Package, error) {
	archive, err := ZIP(file)
	if err != nil {
		return nil, err
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
		return nil, errors.New("not a readable EPUB")
	}
	opf := container.Rootfiles[0].FullPath
	var pkg struct {
		Identifiers []struct {
			Scheme string `xml:"scheme,attr"`
			Value  string `xml:",chardata"`
		} `xml:"metadata>identifier"`
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
		Guide []struct {
			Type string `xml:"type,attr"`
			Href string `xml:"href,attr"`
		} `xml:"guide>reference"`
	}
	if err := decode(files[opf], &pkg); err != nil {
		return nil, errors.New("not a readable EPUB")
	}
	result := &Package{files: files}
	for _, id := range pkg.Identifiers {
		result.Identifiers = append(result.Identifiers, Identifier{Scheme: strings.TrimSpace(id.Scheme), Value: strings.TrimSpace(id.Value)})
	}
	dir := path.Dir(opf)
	name := func(href string) string { return path.Join(dir, unescapeHref(href)) }
	hrefs := map[string]string{}
	for _, item := range pkg.Manifest {
		hrefs[item.ID] = item.Href
	}
	guide := map[string]string{}
	for _, reference := range pkg.Guide {
		guide[name(reference.Href)] = strings.ToLower(reference.Type)
	}
	for _, ref := range pkg.Spine {
		if href, ok := hrefs[ref.IDRef]; ok {
			full := name(href)
			if files[full] != nil {
				result.Spine = append(result.Spine, Document{Name: full, Guide: guide[full]})
			}
		}
	}
	return result, nil
}

// Text is what a reader sees in one document, with a line break after each block so that nothing runs across
// paragraphs, plus the structural labels (epub:type, role) the document carries.
type Text struct {
	Body  string
	Types []string
}

var blocks = map[string]bool{"p": true, "div": true, "br": true, "li": true, "tr": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "section": true, "blockquote": true}

// Text reads one document. At most limit bytes of the file are looked at.
func (p *Package) Text(document Document, limit int64) (Text, error) {
	entry := p.files[document.Name]
	if entry == nil {
		return Text{}, errors.New("missing document")
	}
	reader, err := entry.Open()
	if err != nil {
		return Text{}, err
	}
	defer reader.Close()
	decoder := xml.NewDecoder(io.LimitReader(reader, limit))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	var body strings.Builder
	var types []string
	skip := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if body.Len() == 0 {
				return Text{}, err
			}
			break // a broken tail should not hide the text before it
		}
		switch t := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if name == "head" || name == "script" || name == "style" {
				skip++
			}
			for _, attr := range t.Attr {
				if key := strings.ToLower(attr.Name.Local); key == "type" || key == "role" {
					types = append(types, strings.Fields(strings.ToLower(attr.Value))...)
				}
			}
		case xml.EndElement:
			name := strings.ToLower(t.Name.Local)
			if name == "head" || name == "script" || name == "style" {
				if skip > 0 {
					skip--
				}
			} else if skip == 0 && blocks[name] {
				body.WriteByte('\n')
			}
		case xml.CharData:
			if skip == 0 {
				body.WriteString(string(t))
			}
		}
	}
	return Text{Body: body.String(), Types: types}, nil
}

// unescapeHref resolves the percent-escapes a package document may use in an href.
// A hand-written list of six escapes silently dropped every spine item whose name
// used another one (%23, %3A, %C3%A9, ...), and its text went unread.
func unescapeHref(href string) string {
	if i := strings.IndexAny(href, "#?"); i >= 0 {
		href = href[:i]
	}
	if decoded, err := url.PathUnescape(href); err == nil {
		return decoded
	}
	return href
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
