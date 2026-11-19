package webpub

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"net/url"
	"path"
	"strings"
)

// tocHref resolves a navigation link against the document that wrote it. A
// fragment-only href ("#part1") points inside that same document, so it must
// resolve to the navigation document itself, not to its folder.
func tocHref(document, href string, files map[string]*zip.File) string {
	if href == "" {
		return ""
	}
	fragment := ""
	if i := strings.IndexByte(href, '#'); i >= 0 {
		href, fragment = href[:i], href[i:]
	}
	if unescaped, err := url.PathUnescape(href); err == nil {
		href = unescaped
	}
	resolved := document
	if href != "" {
		resolved = path.Clean(path.Join(path.Dir(document), href))
	}
	if _, ok := files[resolved]; !ok {
		return ""
	}
	return resolved + fragment
}

// ncxTOC reads an EPUB 2 navMap.
func ncxTOC(file *zip.File, document string, files map[string]*zip.File) []Link {
	type point struct {
		Label struct {
			Text string `xml:"text"`
		} `xml:"navLabel"`
		Content struct {
			Src string `xml:"src,attr"`
		} `xml:"content"`
		Children []point `xml:"navPoint"`
	}
	var ncx struct {
		Points []point `xml:"navMap>navPoint"`
	}
	if readXML(file, &ncx) != nil {
		return nil
	}
	var convert func([]point) []Link
	convert = func(points []point) []Link {
		var out []Link
		for _, p := range points {
			link := Link{Title: strings.TrimSpace(p.Label.Text), Href: tocHref(document, p.Content.Src, files), Children: convert(p.Children)}
			if link.Href != "" || len(link.Children) > 0 {
				out = append(out, link)
			}
		}
		return out
	}
	return convert(ncx.Points)
}

// navigationTOC reads the <nav epub:type="toc"> list of an EPUB 3 navigation
// document by walking its tokens, since the markup is XHTML and nests freely.
func navigationTOC(file *zip.File, document string, files map[string]*zip.File) []Link {
	if file == nil || file.UncompressedSize64 > maximumDocumentBytes {
		return nil
	}
	handle, err := file.Open()
	if err != nil {
		return nil
	}
	defer handle.Close()
	decoder := xml.NewDecoder(io.LimitReader(handle, maximumDocumentBytes))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var root []Link
	// stack holds the list currently being filled, as pointers into the tree.
	stack := []*[]Link{&root}
	var current *Link
	inTOC, depthInTOC := false, 0
	var label strings.Builder
	var href string
	// depth counts elements inside the nav; collectAt is the depth of the element
	// whose label is being read. Tracking the anchor instead of a boolean keeps
	// inline markup (<span>, <em>) inside a label instead of ending it early.
	depth, collectAt := 0, 0
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			name := t.Name.Local
			if name == "nav" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "type" && strings.Contains(attr.Value, "toc") {
						inTOC, depthInTOC = true, 0
					}
				}
				continue
			}
			if !inTOC {
				continue
			}
			depth++
			switch name {
			case "ol":
				depthInTOC++
				if depthInTOC > 1 && current != nil {
					stack = append(stack, &current.Children)
				}
			case "li":
				*stack[len(stack)-1] = append(*stack[len(stack)-1], Link{})
				list := *stack[len(stack)-1]
				current = &list[len(list)-1]
			case "a", "span":
				if current != nil && collectAt == 0 {
					collectAt = depth
					label.Reset()
					href = ""
					for _, attr := range t.Attr {
						if attr.Name.Local == "href" {
							href = attr.Value
						}
					}
				}
			}
		case xml.CharData:
			if collectAt != 0 {
				label.Write(t)
			}
		case xml.EndElement:
			name := t.Name.Local
			if !inTOC {
				continue
			}
			if name == "nav" {
				return prune(root, document, files)
			}
			if collectAt != 0 && depth == collectAt && current != nil {
				current.Title = strings.Join(strings.Fields(label.String()), " ")
				current.Href = href
				collectAt = 0
			}
			if name == "ol" {
				if depthInTOC > 1 && len(stack) > 1 {
					stack = stack[:len(stack)-1]
				}
				depthInTOC--
			}
			if depth > 0 {
				depth--
			}
		}
	}
	return prune(root, document, files)
}

// prune resolves hrefs and drops entries with neither a target nor children.
func prune(links []Link, document string, files map[string]*zip.File) []Link {
	var out []Link
	for _, link := range links {
		link.Href = tocHref(document, link.Href, files)
		link.Children = prune(link.Children, document, files)
		if link.Href != "" || len(link.Children) > 0 {
			out = append(out, link)
		}
	}
	return out
}
