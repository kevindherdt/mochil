// Package webpub turns an EPUB into a Readium Web Publication Manifest so the
// optional Readium reader can open books the server already stores. It reads
// only the package document and navigation; chapters are served untouched.
package webpub

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"path"
	"strings"
)

const (
	maximumDocumentBytes = 4 << 20
	maximumItems         = 20000
	// A position stands for about 1 KiB of chapter text, so this allows books of roughly 100 MB of text. Archive
	// metadata can claim any size, so the budget is checked before anything is allocated from it.
	maximumPositions  = 100000
	maximumEntryBytes = int64(1) << 32
)

// ErrTooLarge means a publication declares more content than positions are generated for.
var ErrTooLarge = errors.New("webpub: publication is too large to number")

type Link struct {
	Href       string         `json:"href"`
	Type       string         `json:"type,omitempty"`
	Title      string         `json:"title,omitempty"`
	Rel        []string       `json:"rel,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Children   []Link         `json:"children,omitempty"`
}

type Manifest struct {
	Context      []string `json:"@context"`
	Metadata     Metadata `json:"metadata"`
	Links        []Link   `json:"links"`
	ReadingOrder []Link   `json:"readingOrder"`
	Resources    []Link   `json:"resources"`
	TOC          []Link   `json:"toc,omitempty"`
	sizes        map[string]int64
	pages        map[string]string // generated page href -> image path
}

type Metadata struct {
	Identifier      string   `json:"identifier,omitempty"`
	Title           string   `json:"title"`
	Author          []string `json:"author,omitempty"`
	Language        []string `json:"language,omitempty"`
	ReadingProgress string   `json:"readingProgression,omitempty"`
	Layout          string   `json:"-"`
	Presentation    any      `json:"presentation,omitempty"`
	ConformsTo      string   `json:"conformsTo"`
}

type packageDocument struct {
	Metadata struct {
		Identifiers []string `xml:"identifier"`
		Titles      []string `xml:"title"`
		Creators    []string `xml:"creator"`
		Languages   []string `xml:"language"`
		Meta        []struct {
			Property string `xml:"property,attr"`
			Name     string `xml:"name,attr"`
			Content  string `xml:"content,attr"`
			Value    string `xml:",chardata"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Manifest []struct {
		ID         string `xml:"id,attr"`
		Href       string `xml:"href,attr"`
		MediaType  string `xml:"media-type,attr"`
		Properties string `xml:"properties,attr"`
		Overlay    string `xml:"media-overlay,attr"`
	} `xml:"manifest>item"`
	Spine struct {
		Direction string `xml:"page-progression-direction,attr"`
		TOC       string `xml:"toc,attr"`
		Items     []struct {
			IDRef  string `xml:"idref,attr"`
			Linear string `xml:"linear,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

// Build reads the EPUB in reader and returns its manifest. Every href in the
// result is relative to the archive root, so a server can expose entries at
// <base>/<href> and serve the manifest next to them.
func Build(reader io.ReaderAt, size int64) (*Manifest, error) {
	_, files, opfPath, pkg, err := openPackage(reader, size)
	if err != nil {
		return nil, err
	}
	base := path.Dir(opfPath)
	// One resolver for the whole package: Build, the media overlays and the
	// text-only copy must agree on what an href names.
	resolve := func(href string) string { return resolveIn(base, href) }

	manifest := &Manifest{Context: []string{"https://readium.org/webpub-manifest/context.jsonld"}, sizes: map[string]int64{}, pages: map[string]string{}}
	manifest.Metadata.ConformsTo = "https://readium.org/webpub-manifest/profiles/epub"
	manifest.Metadata.Title = first(pkg.Metadata.Titles)
	manifest.Metadata.Identifier = first(pkg.Metadata.Identifiers)
	manifest.Metadata.Author = nonEmpty(pkg.Metadata.Creators)
	manifest.Metadata.Language = nonEmpty(pkg.Metadata.Languages)
	switch strings.ToLower(pkg.Spine.Direction) {
	case "rtl", "ltr":
		manifest.Metadata.ReadingProgress = strings.ToLower(pkg.Spine.Direction)
	}
	for _, meta := range pkg.Metadata.Meta {
		if meta.Property == "rendition:layout" && strings.TrimSpace(meta.Value) == "pre-paginated" {
			manifest.Metadata.Presentation = map[string]any{"layout": "fixed"}
		}
	}

	byID := make(map[string]int, len(pkg.Manifest))
	var navPath, ncxPath string
	for i, item := range pkg.Manifest {
		byID[item.ID] = i
		href := resolve(item.Href)
		if _, ok := files[href]; !ok {
			continue
		}
		properties := strings.Fields(item.Properties)
		if contains(properties, "nav") {
			navPath = href
		}
		if item.MediaType == "application/x-dtbncx+xml" || item.ID == pkg.Spine.TOC {
			ncxPath = href
		}
	}
	// Bounded work for the image-page probe across the whole spine.
	probeBudget := int64(maximumSpineProbe)
	inSpine := map[string]bool{}
	split := map[string]string{}
	generated := map[string]bool{}
	names := nameSet(files)
	for _, ref := range pkg.Spine.Items {
		i, ok := byID[ref.IDRef]
		if !ok {
			continue
		}
		item := pkg.Manifest[i]
		href := resolve(item.Href)
		if _, ok := files[href]; !ok {
			continue
		}
		link := Link{Href: href, Type: item.MediaType}
		manifest.sizes[href] = min(int64(min(files[href].UncompressedSize64, uint64(maximumEntryBytes))), maximumEntryBytes)
		if ref.Linear == "no" {
			link.Properties = map[string]any{"linear": "no"}
		}
		inSpine[href] = true
		if generated[href] {
			// A malformed spine can repeat an item; its pages already exist.
			continue
		}
		if len(manifest.ReadingOrder) < maximumVirtualPages {
			if pages := splitImageDocument(files[href], href, len(manifest.ReadingOrder), names, manifest, &probeBudget); len(pages) > 0 && len(manifest.ReadingOrder)+len(pages) <= maximumVirtualPages {
				generated[href] = true
				split[href] = pages[0].Href
				manifest.ReadingOrder = append(manifest.ReadingOrder, pages...)
				continue
			}
		}
		manifest.ReadingOrder = append(manifest.ReadingOrder, link)
	}
	if len(manifest.ReadingOrder) == 0 {
		return nil, errors.New("webpub: no readable spine items")
	}
	for _, item := range pkg.Manifest {
		href := resolve(item.Href)
		if _, ok := files[href]; !ok || inSpine[href] {
			continue
		}
		link := Link{Href: href, Type: item.MediaType}
		for _, property := range strings.Fields(item.Properties) {
			if property == "cover-image" {
				link.Rel = append(link.Rel, "cover")
			}
		}
		manifest.Resources = append(manifest.Resources, link)
	}
	switch {
	case navPath != "":
		manifest.TOC = navigationTOC(files[navPath], navPath, files)
	case ncxPath != "":
		manifest.TOC = ncxTOC(files[ncxPath], ncxPath, files)
	}
	if len(split) > 0 {
		manifest.TOC = remap(manifest.TOC, split)
	}
	return manifest, nil
}

// remap points navigation entries of a split document at its first page.
func remap(links []Link, split map[string]string) []Link {
	for i := range links {
		document := links[i].Href
		if cut := strings.IndexByte(document, '#'); cut >= 0 {
			document = document[:cut]
		}
		if first, ok := split[document]; ok {
			links[i].Href = first
		}
		links[i].Children = remap(links[i].Children, split)
	}
	return links
}

func nameSet(files map[string]*zip.File) map[string]bool {
	names := make(map[string]bool, len(files))
	for name := range files {
		names[name] = true
	}
	return names
}

// splitImageDocument returns one generated link per image when the document is
// a long run of page images, and nil otherwise.
func splitImageDocument(file *zip.File, href string, specIndex int, names map[string]bool, manifest *Manifest, budget *int64) []Link {
	if file == nil || *budget <= 0 || file.UncompressedSize64 > maximumPageBody || !strings.HasSuffix(strings.ToLower(href), "html") {
		return nil
	}
	body, err := readZipBytes(file)
	// Every spine item is inflated to find out whether it is an image-only page, so
	// the total has to be bounded: a crafted book with thousands of chapters would
	// otherwise cost gigabytes per manifest build, and the manifest is built per
	// page request.
	*budget -= int64(len(body))
	if err != nil {
		return nil
	}
	images := imageOnlyPages(body, path.Dir(href), names)
	if images == nil {
		return nil
	}
	links := make([]Link, 0, len(images))
	for n, image := range images {
		virtual := virtualHref(specIndex, n+1)
		manifest.pages[virtual] = image
		manifest.sizes[virtual] = bytesPerPosition
		links = append(links, Link{Href: virtual, Type: "application/xhtml+xml"})
	}
	return links
}

// Open returns one archive entry's reader and size, or false when absent.
func Open(reader io.ReaderAt, size int64, name string) (io.ReadCloser, int64, bool) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, 0, false
	}
	name = path.Clean(strings.TrimPrefix(name, "/"))
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return nil, 0, false
		}
		return handle, int64(file.UncompressedSize64), true
	}
	return nil, 0, false
}

func readXML(file *zip.File, into any) error {
	if file == nil {
		return errors.New("webpub: file is missing")
	}
	if file.UncompressedSize64 > maximumDocumentBytes {
		return errors.New("webpub: document is too large")
	}
	handle, err := file.Open()
	if err != nil {
		return err
	}
	defer handle.Close()
	decoder := xml.NewDecoder(io.LimitReader(handle, maximumDocumentBytes))
	decoder.Strict = false
	return decoder.Decode(into)
}

func first(values []string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func nonEmpty(values []string) []string {
	var out []string
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// PositionList is the Readium "positions" service: one locator per roughly
// 1,024 bytes of chapter text, which the navigator uses for page-style
// progress and for resolving a fraction to a place in the book.
type PositionList struct {
	Total     int              `json:"total"`
	Positions []map[string]any `json:"positions"`
}

const bytesPerPosition = 1024

func (m *Manifest) Positions() (PositionList, error) {
	var total int64
	counts := make([]int64, len(m.ReadingOrder))
	for i, link := range m.ReadingOrder {
		counts[i] = max(1, (min(max(m.sizes[link.Href], 0), maximumEntryBytes)+bytesPerPosition-1)/bytesPerPosition)
		total += counts[i]
		if total > maximumPositions {
			return PositionList{}, ErrTooLarge
		}
	}
	list := PositionList{Positions: make([]map[string]any, 0, total)}
	var number int64
	for i, link := range m.ReadingOrder {
		for j := int64(0); j < counts[i]; j++ {
			number++
			list.Positions = append(list.Positions, map[string]any{
				"href": link.Href, "type": link.Type,
				"locations": map[string]any{
					"position":         number,
					"progression":      float64(j) / float64(counts[i]),
					"totalProgression": float64(number-1) / float64(total),
				},
			})
		}
	}
	list.Total = len(list.Positions)
	return list, nil
}

func readZipBytes(file *zip.File) ([]byte, error) {
	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	return io.ReadAll(io.LimitReader(handle, maximumPageBody))
}

// openPackage opens the archive and parses its package document.
func openPackage(reader io.ReaderAt, size int64) (*zip.Reader, map[string]*zip.File, string, *packageDocument, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, nil, "", nil, err
	}
	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[file.Name] = file
	}
	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := readXML(files["META-INF/container.xml"], &container); err != nil || len(container.Rootfiles) == 0 {
		return nil, nil, "", nil, errors.New("webpub: missing container rootfile")
	}
	opfPath := container.Rootfiles[0].Path
	pkg := &packageDocument{}
	if err := readXML(files[opfPath], pkg); err != nil {
		return nil, nil, "", nil, err
	}
	if len(pkg.Spine.Items) == 0 || len(pkg.Spine.Items) > maximumItems || len(pkg.Manifest) > maximumItems {
		return nil, nil, "", nil, errors.New("webpub: spine is empty or too large")
	}
	return archive, files, opfPath, pkg, nil
}
