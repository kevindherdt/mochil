// Package opf writes the metadata.opf sidecar that Calibre keeps beside each book, from a book's fields. Readers such as
// Calibre and Mochil' own Calibre libraries understand it, and writing it never touches the book file.
package opf

import (
	"bytes"
	"crypto/sha1"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Metadata is what goes into the sidecar. Rating is 0 to 10, as Calibre stores it.
type Metadata struct {
	Title       string
	TitleSort   string
	Authors     []string
	AuthorSort  string
	Series      string
	SeriesIndex *float64
	Tags        []string
	Publisher   string
	Published   string
	Language    string
	Rating      int
	Description string
	Identifiers map[string]string
}

func escape(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}

// attribute escapes a value for use inside double quotes.
func attribute(value string) string {
	return strings.ReplaceAll(escape(value), `"`, "&#34;")
}

// Generate returns the sidecar. The same fields always give the same bytes, so rewriting an unchanged book changes
// nothing on disk.
func Generate(meta Metadata) ([]byte, error) {
	title := strings.TrimSpace(meta.Title)
	if title == "" {
		return nil, errors.New("a sidecar needs a title")
	}
	var out bytes.Buffer
	out.WriteString(`<?xml version='1.0' encoding='utf-8'?>` + "\n")
	out.WriteString(`<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uuid_id">` + "\n")
	out.WriteString(`  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">` + "\n")
	fmt.Fprintf(&out, "    <dc:identifier opf:scheme=\"uuid\" id=\"uuid_id\">%s</dc:identifier>\n", stableUUID(meta))
	fmt.Fprintf(&out, "    <dc:title>%s</dc:title>\n", escape(title))
	fileAs := strings.Split(strings.TrimSpace(meta.AuthorSort), " & ")
	for index, author := range meta.Authors {
		if author = strings.TrimSpace(author); author == "" {
			continue
		}
		as := ""
		if index < len(fileAs) && strings.TrimSpace(fileAs[index]) != "" {
			as = fmt.Sprintf(` opf:file-as="%s"`, attribute(strings.TrimSpace(fileAs[index])))
		}
		fmt.Fprintf(&out, "    <dc:creator opf:role=\"aut\"%s>%s</dc:creator>\n", as, escape(author))
	}
	if value := strings.TrimSpace(meta.Publisher); value != "" {
		fmt.Fprintf(&out, "    <dc:publisher>%s</dc:publisher>\n", escape(value))
	}
	if value := strings.TrimSpace(meta.Published); value != "" {
		fmt.Fprintf(&out, "    <dc:date>%s</dc:date>\n", escape(value))
	}
	if value := strings.TrimSpace(meta.Description); value != "" {
		fmt.Fprintf(&out, "    <dc:description>%s</dc:description>\n", escape(value))
	}
	if value := strings.TrimSpace(meta.Language); value != "" {
		fmt.Fprintf(&out, "    <dc:language>%s</dc:language>\n", escape(value))
	}
	schemes := make([]string, 0, len(meta.Identifiers))
	for scheme := range meta.Identifiers {
		schemes = append(schemes, scheme)
	}
	sort.Strings(schemes)
	for _, scheme := range schemes {
		id := strings.TrimSpace(meta.Identifiers[scheme])
		if scheme = strings.ToLower(strings.TrimSpace(scheme)); scheme == "" || id == "" || scheme == "uuid" {
			continue
		}
		fmt.Fprintf(&out, "    <dc:identifier opf:scheme=\"%s\">%s</dc:identifier>\n", attribute(scheme), escape(id))
	}
	for _, tag := range meta.Tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			fmt.Fprintf(&out, "    <dc:subject>%s</dc:subject>\n", escape(tag))
		}
	}
	if series := strings.TrimSpace(meta.Series); series != "" {
		fmt.Fprintf(&out, "    <meta name=\"calibre:series\" content=\"%s\"/>\n", attribute(series))
		if meta.SeriesIndex != nil {
			fmt.Fprintf(&out, "    <meta name=\"calibre:series_index\" content=\"%s\"/>\n", strconv.FormatFloat(*meta.SeriesIndex, 'f', -1, 64))
		}
	}
	if meta.Rating > 0 && meta.Rating <= 10 {
		fmt.Fprintf(&out, "    <meta name=\"calibre:rating\" content=\"%d.0\"/>\n", meta.Rating)
	}
	if value := strings.TrimSpace(meta.TitleSort); value != "" {
		fmt.Fprintf(&out, "    <meta name=\"calibre:title_sort\" content=\"%s\"/>\n", attribute(value))
	}
	out.WriteString("  </metadata>\n</package>\n")
	return out.Bytes(), nil
}

// stableUUID makes a repeatable identifier from the book's own fields, so the file is valid without a library database.
func stableUUID(meta Metadata) string {
	sum := sha1.Sum([]byte(strings.ToLower(meta.Title + "\x00" + strings.Join(meta.Authors, "\x00") + "\x00" + meta.Series)))
	b := sum[:16]
	b[6] = b[6]&0x0f | 0x50
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ErrExists means a sidecar is already there and was left alone.
var ErrExists = errors.New("a sidecar already exists")

// SidecarPath is where a book's sidecar goes: metadata.opf when the book is alone in its folder (Calibre's layout), or
// <book name>.opf when the folder holds several books, so they do not share one file.
func SidecarPath(bookPath string) string {
	dir := filepath.Dir(bookPath)
	entries, err := os.ReadDir(dir)
	others := 0
	if err == nil {
		for _, entry := range entries {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if !entry.IsDir() && entry.Name() != filepath.Base(bookPath) && (ext == ".epub" || ext == ".pdf" || ext == ".mobi" || ext == ".azw3" || ext == ".cbz") {
				others++
			}
		}
	}
	if others == 0 {
		return filepath.Join(dir, "metadata.opf")
	}
	return strings.TrimSuffix(bookPath, filepath.Ext(bookPath)) + ".opf"
}

// WriteBeside writes the sidecar for a book that already exists on disk. It never replaces a file: an existing sidecar
// returns ErrExists unless overwrite is set, and an overwrite is made through a temporary file so a failure cannot leave
// half a sidecar.
func WriteBeside(bookPath string, meta Metadata, overwrite bool) (string, error) {
	if info, err := os.Lstat(bookPath); err != nil || !info.Mode().IsRegular() {
		return "", errors.New("the book is missing or is not a regular file")
	}
	content, err := Generate(meta)
	if err != nil {
		return "", err
	}
	target := SidecarPath(bookPath)
	if !overwrite {
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			return target, ErrExists
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(content)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(target)
			return "", errors.Join(writeErr, closeErr)
		}
		return target, nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".opf-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(temporary.Name(), 0o644); err != nil {
		return "", err
	}
	return target, os.Rename(temporary.Name(), target)
}
