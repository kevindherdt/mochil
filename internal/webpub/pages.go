package webpub

import (
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// Some converters (Calibre's PDF reflow, for one) put every scanned page of a
// book into a single HTML document of <p><img></p> blocks. Laid out as columns,
// hundreds of full-size images in one document render slowly and show blank
// pages while they decode. Such a document is split into one small virtual
// page per image so each loads when it is reached.
const (
	virtualPrefix   = "_bobo/page/"
	minimumImages   = 8
	maximumLoose    = 160 // visible characters allowed besides the images
	maximumPageBody = 2 << 20
	// Total bytes one manifest build may inflate while looking for image-only pages.
	maximumSpineProbe = 16 << 20
)

var (
	imageSource  = regexp.MustCompile(`(?is)<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']`)
	markup       = regexp.MustCompile(`(?s)<[^>]*>`)
	headBlock    = regexp.MustCompile(`(?is)<head\b.*?</head>`)
	commentBlock = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// maximumVirtualPages bounds how many pages one archive may generate in total,
// so a spine that repeats an image-heavy document cannot multiply the work.
const maximumVirtualPages = 5000

// imageOnlyPages returns the image paths of an image-only document, in order.
func imageOnlyPages(body []byte, base string, files map[string]bool) []string {
	// Comments never render, so an <img> inside one must not become a page.
	source := commentBlock.ReplaceAllString(string(body), "")
	matches := imageSource.FindAllStringSubmatch(source, -1)
	if len(matches) < minimumImages {
		return nil
	}
	text := markup.ReplaceAllString(headBlock.ReplaceAllString(source, ""), " ")
	if len(strings.Join(strings.Fields(html.UnescapeString(text)), " ")) > maximumLoose {
		return nil
	}
	pages := make([]string, 0, len(matches))
	for _, match := range matches {
		href := match[1]
		if unescaped, err := url.PathUnescape(href); err == nil {
			href = unescaped
		}
		resolved := path.Clean(path.Join(base, href))
		if !files[resolved] {
			return nil
		}
		pages = append(pages, resolved)
	}
	return pages
}

func virtualHref(specIndex, page int) string {
	return fmt.Sprintf("%s%d-%d.xhtml", virtualPrefix, specIndex, page)
}

// IsVirtual reports whether href names a generated page.
func IsVirtual(href string) bool { return strings.HasPrefix(href, virtualPrefix) }

// VirtualPage renders the XHTML of a generated page, or false when unknown.
func (m *Manifest) VirtualPage(href string) (string, bool) {
	image, ok := m.pages[href]
	if !ok {
		return "", false
	}
	segments := strings.Split(image, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	source := "../../" + strings.Join(segments, "/")
	return `<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Page</title>
<style>html,body{margin:0;padding:0;height:100%}body{display:flex;align-items:center;justify-content:center}img{display:block;max-width:100%;max-height:100vh;object-fit:contain}</style>
</head><body><img src="` + html.EscapeString(source) + `" alt=""/></body></html>`, true
}
