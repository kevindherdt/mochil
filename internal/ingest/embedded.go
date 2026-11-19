package ingest

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"path"
	"strconv"
	"strings"
)

const maximumOPFBytes = 4 << 20

// Embedded is what a publication says about itself in its package document.
type Embedded struct {
	Title       string
	Authors     []string
	Series      string
	SeriesIndex *float64
	Language    string
	Identifiers map[string]string
}

type opfPackage struct {
	Metadata struct {
		Titles      []string `xml:"title"`
		Creators    []string `xml:"creator"`
		Languages   []string `xml:"language"`
		Identifiers []struct {
			Scheme string `xml:"scheme,attr"`
			Value  string `xml:",chardata"`
		} `xml:"identifier"`
		Metas []struct {
			Name     string `xml:"name,attr"`
			Content  string `xml:"content,attr"`
			Property string `xml:"property,attr"`
			ID       string `xml:"id,attr"`
			Refines  string `xml:"refines,attr"`
			Value    string `xml:",chardata"`
		} `xml:"meta"`
	} `xml:"metadata"`
}

// ReadEmbedded reads title, authors, series and identifiers from an EPUB without changing it.
func ReadEmbedded(file string) (Embedded, error) {
	archive, err := zip.OpenReader(file)
	if err != nil {
		return Embedded{}, err
	}
	defer archive.Close()
	opfPath, err := rootfile(archive)
	if err != nil {
		return Embedded{}, err
	}
	raw, err := readZipEntry(archive, opfPath)
	if err != nil {
		return Embedded{}, err
	}
	return ParseOPF(raw)
}

// ParseOPF understands Calibre's series meta tags and EPUB 3 collections.
func ParseOPF(raw []byte) (Embedded, error) {
	var pkg opfPackage
	if err := xml.Unmarshal(raw, &pkg); err != nil {
		return Embedded{}, err
	}
	out := Embedded{Identifiers: map[string]string{}}
	if len(pkg.Metadata.Titles) > 0 {
		out.Title = strings.TrimSpace(pkg.Metadata.Titles[0])
	}
	for _, creator := range pkg.Metadata.Creators {
		if name := strings.TrimSpace(creator); name != "" {
			out.Authors = append(out.Authors, name)
		}
	}
	if len(pkg.Metadata.Languages) > 0 {
		out.Language = strings.TrimSpace(pkg.Metadata.Languages[0])
	}
	for _, id := range pkg.Metadata.Identifiers {
		value := strings.TrimSpace(id.Value)
		scheme := strings.ToLower(strings.TrimSpace(id.Scheme))
		if lower := strings.ToLower(value); strings.HasPrefix(lower, "urn:isbn:") {
			scheme, value = "isbn", value[len("urn:isbn:"):]
		}
		if scheme != "" && value != "" && scheme != "uuid" {
			out.Identifiers[scheme] = value
		}
	}
	collections := map[string]string{}
	indexes := map[string]string{}
	for _, meta := range pkg.Metadata.Metas {
		value := strings.TrimSpace(meta.Value)
		switch {
		case meta.Name == "calibre:series" && meta.Content != "":
			out.Series = strings.TrimSpace(meta.Content)
		case meta.Name == "calibre:series_index":
			out.SeriesIndex = parseIndex(meta.Content)
		case meta.Property == "belongs-to-collection" && value != "" && meta.Refines == "":
			collections["#"+meta.ID] = value
			if out.Series == "" {
				out.Series = value
			}
		case meta.Property == "belongs-to-collection" && value != "":
			collections[meta.Refines] = value
			if out.Series == "" {
				out.Series = value
			}
		case meta.Property == "group-position":
			indexes[meta.Refines] = value
		}
	}
	if out.SeriesIndex == nil {
		for ref, value := range indexes {
			if _, ok := collections[ref]; ok {
				out.SeriesIndex = parseIndex(value)
			}
		}
	}
	if len(out.Identifiers) == 0 {
		out.Identifiers = nil
	}
	return out, nil
}

func parseIndex(value string) *float64 {
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	// NaN and ±Inf parse as floats and then cannot be encoded as JSON, which
	// turned one crafted EPUB into an import listing with no valid body at all.
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return nil
	}
	return &number
}

func rootfile(archive *zip.ReadCloser) (string, error) {
	raw, err := readZipEntry(archive, "META-INF/container.xml")
	if err != nil {
		return "", err
	}
	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(raw, &container); err != nil || len(container.Rootfiles) == 0 {
		return "", errors.New("the book has no package document")
	}
	return path.Clean(container.Rootfiles[0].Path), nil
}

func readZipEntry(archive *zip.ReadCloser, name string) ([]byte, error) {
	for _, file := range archive.File {
		if file.Name == name {
			reader, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer reader.Close()
			raw, err := io.ReadAll(io.LimitReader(reader, maximumOPFBytes+1))
			if err != nil {
				return nil, err
			}
			if len(raw) > maximumOPFBytes {
				return nil, errors.New("the package document is too large")
			}
			return raw, nil
		}
	}
	return nil, errors.New(name + " is missing")
}

var placeholderValues = map[string]bool{"unknown": true, "untitled": true, "unknown author": true, "various": false, "": true, "author": true, "anonymous": true}

// usable reports whether an embedded value says something. Calibre-less tools often leave placeholders behind.
func usable(value string) bool {
	return !placeholderValues[strings.ToLower(strings.TrimSpace(value))]
}

// Combine merges what the file says about itself with what its folders and name suggest. The book's own metadata is
// trusted for each field it fills in; the path fills the gaps. Disagreements are recorded for review, not decided silently.
func Combine(fromPath Proposal, embedded Embedded) Proposal {
	out := fromPath
	out.Evidence = append([]Evidence(nil), fromPath.Evidence...)
	if usable(embedded.Title) {
		if out.Title != "" && !overlapping(out.Title, embedded.Title) {
			out.Conflicts = append(out.Conflicts, "title: the book says \""+embedded.Title+"\", its filename says \""+out.Title+"\"")
		}
		out.Title = embedded.Title
		out.add("title", embedded.Title, "embedded metadata", High)
	}
	var authors []string
	for _, author := range embedded.Authors {
		if usable(author) {
			authors = append(authors, author)
		}
	}
	if len(authors) > 0 {
		// A folder's name is a weak guess at an author (it may be a batch or category folder), so the book's own
		// authors replace it without a fuss; only a folder author the book also names is kept as agreement.
		out.Authors = authors
		out.add("authors", strings.Join(authors, ", "), "embedded metadata", High)
	}
	if usable(embedded.Series) {
		if out.Series != "" && !overlapping(out.Series, embedded.Series) {
			out.Conflicts = append(out.Conflicts, "series: the book says \""+embedded.Series+"\", its folder says \""+out.Series+"\"")
		}
		out.Series = embedded.Series
		out.add("series", embedded.Series, "embedded metadata", High)
	}
	if embedded.SeriesIndex != nil {
		if out.SeriesIndex != nil && *out.SeriesIndex != *embedded.SeriesIndex {
			out.Conflicts = append(out.Conflicts, "number: the book says "+strconv.FormatFloat(*embedded.SeriesIndex, 'f', -1, 64)+", its filename says "+strconv.FormatFloat(*out.SeriesIndex, 'f', -1, 64))
		}
		out.SeriesIndex = embedded.SeriesIndex
		out.add("series_index", strconv.FormatFloat(*embedded.SeriesIndex, 'f', -1, 64), "embedded metadata", High)
	}
	for key, value := range embedded.Identifiers {
		if out.Identifiers == nil {
			out.Identifiers = map[string]string{}
		}
		if _, exists := out.Identifiers[key]; !exists {
			out.Identifiers[key] = value
			out.add("identifier."+key, value, "embedded metadata", High)
		}
	}
	return out
}

// overlapping is true when one title contains the other once case and punctuation are ignored: a filename that drops
// the series prefix does not disagree with the book's full title.
func overlapping(a, b string) bool {
	na, nb := normalize(a), normalize(b)
	// a name that is only punctuation, like "++", has nothing to disagree with
	return na == "" || nb == "" || na == nb || strings.Contains(na, nb) || strings.Contains(nb, na)
}
