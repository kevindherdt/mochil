package booktext

import (
	"io"

	"mochil/internal/webpub"
)

// Chapter is one entry of a book's reading order.
type Chapter struct {
	Index int    `json:"index"`
	Href  string `json:"href"`
	Title string `json:"title,omitempty"`
	// Matter marks front and back matter (copyright pages, "also by" lists) that is not read.
	Matter bool `json:"matter,omitempty"`
}

// Chapters lists a book's reading order with the titles from its table of contents.
func Chapters(manifest *webpub.Manifest) []Chapter {
	titles := map[string]string{}
	var collect func(links []webpub.Link)
	collect = func(links []webpub.Link) {
		for _, link := range links {
			href := link.Href
			for i := 0; i < len(href); i++ {
				if href[i] == '#' {
					href = href[:i]
					break
				}
			}
			if _, seen := titles[href]; !seen {
				titles[href] = link.Title
			}
			collect(link.Children)
		}
	}
	collect(manifest.TOC)
	chapters := make([]Chapter, 0, len(manifest.ReadingOrder))
	for i, link := range manifest.ReadingOrder {
		title := titles[link.Href]
		chapters = append(chapters, Chapter{Index: i, Href: link.Href, Title: title, Matter: IsMatterTitle(title)})
	}
	return chapters
}

// ChapterSentences reads one chapter of an EPUB.
func ChapterSentences(reader io.ReaderAt, size int64, href string) ([]Sentence, bool) {
	handle, _, ok := webpub.Open(reader, size, href)
	if !ok {
		return nil, false
	}
	defer handle.Close()
	return Sentences(io.LimitReader(handle, 8<<20)), true
}
