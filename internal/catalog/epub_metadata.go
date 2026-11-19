package catalog

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type packageXML struct {
	Metadata struct {
		Titles   []string `xml:"title"`
		Creators []struct {
			Name   string `xml:",chardata"`
			FileAs string `xml:"file-as,attr"`
			ID     string `xml:"id,attr"`
		} `xml:"creator"`
		Languages    []string `xml:"language"`
		Descriptions []string `xml:"description"`
		Identifiers  []struct {
			Scheme string `xml:"scheme,attr"`
			Value  string `xml:",chardata"`
		} `xml:"identifier"`
		Subjects   []string `xml:"subject"`
		Publishers []string `xml:"publisher"`
		Dates      []string `xml:"date"`
		Metas      []struct {
			ID       string `xml:"id,attr"`
			Refines  string `xml:"refines,attr"`
			Name     string `xml:"name,attr"`
			Content  string `xml:"content,attr"`
			Property string `xml:"property,attr"`
			Value    string `xml:",chardata"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			Overlay string `xml:"media-overlay,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
}

// epubReadAlong reports whether the book carries its own recorded narration (EPUB 3 Media Overlays,
// as Storyteller makes them): a chapter that names an overlay, or a declared narration duration.
func epubReadAlong(pkg packageXML) bool {
	for _, item := range pkg.Manifest.Items {
		if item.Overlay != "" {
			return true
		}
	}
	for _, meta := range pkg.Metadata.Metas {
		if strings.EqualFold(meta.Property, "media:duration") && strings.TrimSpace(meta.Refines) == "" {
			return true
		}
	}
	return false
}

func epubCreators(pkg packageXML) ([]string, []string) {
	roles, refinedSort := map[string][]string{}, map[string]string{}
	for _, meta := range pkg.Metadata.Metas {
		id := strings.TrimPrefix(meta.Refines, "#")
		switch strings.ToLower(meta.Property) {
		case "role":
			roles[id] = append(roles[id], strings.ToLower(clean(meta.Value)))
		case "file-as":
			refinedSort[id] = clean(meta.Value)
		}
	}
	authors := make([]string, 0, len(pkg.Metadata.Creators))
	authorSort := make([]string, 0, len(pkg.Metadata.Creators))
	for _, creator := range pkg.Metadata.Creators {
		// An element can carry several roles — a Standard Ebooks author is marked aut, dto and wpr —
		// so the creator counts as an author if any role says so, not if the last one says so.
		if declared := roles[creator.ID]; len(declared) > 0 && !slices.Contains(declared, "aut") {
			continue
		}
		name := clean(creator.Name)
		if name == "" {
			continue
		}
		authors = append(authors, name)
		sortName := refinedSort[creator.ID]
		if sortName == "" {
			sortName = clean(creator.FileAs)
		}
		if sortName == "" {
			sortName = name
		}
		authorSort = append(authorSort, sortName)
	}
	if len(authors) == 0 {
		return []string{"Unknown"}, []string{"Unknown"}
	}
	return authors, authorSort
}

func readEPUB(path, rel string, info os.FileInfo) (Book, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return Book{}, err
	}
	defer zr.Close()
	containerFile := zipFile(zr.File, "META-INF/container.xml")
	if containerFile == nil {
		return Book{}, errors.New("missing META-INF/container.xml")
	}
	var container containerXML
	if err := decodeZipXML(containerFile, &container); err != nil || len(container.Rootfiles) == 0 {
		return Book{}, errors.New("invalid EPUB container")
	}
	opf := zipFile(zr.File, container.Rootfiles[0].FullPath)
	if opf == nil {
		return Book{}, errors.New("missing OPF package")
	}
	var pkg packageXML
	if err := decodeZipXML(opf, &pkg); err != nil {
		return Book{}, err
	}
	title := first(pkg.Metadata.Titles)
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	authors, authorSort := epubCreators(pkg)
	series, index := epubSeries(pkg)
	book := Book{ID: bookID(rel), Path: rel, Title: clean(title), Authors: authors, AuthorSort: authorSort, Series: series, SeriesIndex: index, Format: "epub", MediaType: "application/epub+zip",
		Language: clean(first(pkg.Metadata.Languages)), Description: truncate(clean(first(pkg.Metadata.Descriptions)), 4000), Identifier: firstIdentifier(pkg), ReadAlong: epubReadAlong(pkg), Size: info.Size(), Modified: info.ModTime()}
	return withExtraMetadata(book, pkg), nil
}

func firstIdentifier(pkg packageXML) string {
	for _, id := range pkg.Metadata.Identifiers {
		if value := clean(id.Value); value != "" {
			return value
		}
	}
	return ""
}

// withExtraMetadata adds what Calibre-style tools keep beyond the title: tags, publisher, date, rating, identifiers
// and sort names. The first identifier stays in Book.Identifier for older callers.
func withExtraMetadata(book Book, pkg packageXML) Book {
	for _, subject := range pkg.Metadata.Subjects {
		if tag := clean(subject); tag != "" {
			book.Tags = append(book.Tags, tag)
		}
	}
	book.Publisher = clean(first(pkg.Metadata.Publishers))
	book.Published = clean(first(pkg.Metadata.Dates))
	for _, id := range pkg.Metadata.Identifiers {
		scheme, value := strings.ToLower(clean(id.Scheme)), clean(id.Value)
		if lower := strings.ToLower(value); strings.HasPrefix(lower, "urn:isbn:") {
			scheme, value = "isbn", value[len("urn:isbn:"):]
		}
		if scheme == "" || value == "" || scheme == "uuid" || scheme == "calibre" {
			continue
		}
		if book.Identifiers == nil {
			book.Identifiers = map[string]string{}
		}
		if _, exists := book.Identifiers[scheme]; !exists {
			book.Identifiers[scheme] = value
		}
	}
	for _, meta := range pkg.Metadata.Metas {
		switch strings.ToLower(meta.Name) {
		case "calibre:rating":
			if rating, err := strconv.ParseFloat(strings.TrimSpace(meta.Content), 64); err == nil && rating >= 0 && rating <= 10 {
				book.Rating = int(rating)
			}
		case "calibre:title_sort":
			book.TitleSort = clean(meta.Content)
		}
	}
	return book
}

func epubSeries(pkg packageXML) (string, *float64) {
	var calibreSeries string
	var calibreIndex *float64
	type collection struct{ id, name, kind, position string }
	collections := make(map[string]*collection)
	var collectionOrder []string
	for _, meta := range pkg.Metadata.Metas {
		switch strings.ToLower(meta.Name) {
		case "calibre:series":
			calibreSeries = clean(meta.Content)
		case "calibre:series_index":
			calibreIndex = parseSeriesIndex(meta.Content)
		}
		if meta.Property == "belongs-to-collection" && meta.ID != "" {
			collections[meta.ID] = &collection{id: meta.ID, name: clean(meta.Value)}
			collectionOrder = append(collectionOrder, meta.ID)
		}
	}
	for _, meta := range pkg.Metadata.Metas {
		collection := collections[strings.TrimPrefix(meta.Refines, "#")]
		if collection == nil {
			continue
		}
		switch meta.Property {
		case "collection-type":
			collection.kind = clean(meta.Value)
		case "group-position":
			collection.position = clean(meta.Value)
		}
	}
	for _, id := range collectionOrder {
		collection := collections[id]
		if collection.kind == "series" && collection.name != "" {
			return collection.name, parseSeriesIndex(collection.position)
		}
	}
	return calibreSeries, calibreIndex
}

func zipFile(files []*zip.File, name string) *zip.File {
	want := filepath.ToSlash(name)
	for _, file := range files {
		if filepath.ToSlash(file.Name) == want {
			return file
		}
	}
	return nil
}

func decodeZipXML(file *zip.File, target any) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return xml.NewDecoder(io.LimitReader(reader, 8<<20)).Decode(target)
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func clean(value string) string { return strings.Join(strings.Fields(value), " ") }

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func parseSeriesIndex(value string) *float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return nil
	}
	return &parsed
}
