package podcast

import (
	"encoding/xml"
	"fmt"
)

// opmlDocument and opmlOutline are the minimal OPML 2.0 shape podcast clients
// (Apple Podcasts, Overcast, AntennaPod, Pocket Casts, ...) actually read and
// write: a flat list of <outline type="rss"> entries. Nested folders in an
// imported file are flattened — every outline with an xmlUrl is subscribed,
// regardless of nesting, since Mochil has no folder concept for podcasts.
type opmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    opmlBody `xml:"body"`
}

type opmlHead struct {
	Title string `xml:"title"`
}

type opmlBody struct {
	Outlines []opmlOutline `xml:"outline"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string        `xml:"htmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline"`
}

// ExportOPML renders every subscribed feed as an OPML 2.0 document, the
// format every major podcast app imports and exports.
func (s *Store) ExportOPML() []byte {
	feeds := s.Feeds()
	doc := opmlDocument{Version: "2.0", Head: opmlHead{Title: "Mochil podcast subscriptions"}}
	for _, feed := range feeds {
		title := feed.Title
		if title == "" {
			title = feed.URL
		}
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{
			Text: title, Title: title, Type: "rss", XMLURL: feed.URL,
		})
	}
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		// doc is a fixed, always-marshalable shape; a failure here would be a
		// programming error, not a runtime one.
		panic(fmt.Errorf("marshal OPML: %w", err))
	}
	return append([]byte(xml.Header), out...)
}

// feedURLsFromOPML walks every outline (including nested ones) and collects
// each xmlUrl attribute, in document order, deduplicated.
func feedURLsFromOPML(data []byte) ([]string, error) {
	var doc opmlDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("not a valid OPML file: %w", err)
	}
	seen := map[string]bool{}
	var urls []string
	var walk func([]opmlOutline)
	walk = func(outlines []opmlOutline) {
		for _, outline := range outlines {
			if outline.XMLURL != "" && !seen[outline.XMLURL] {
				seen[outline.XMLURL] = true
				urls = append(urls, outline.XMLURL)
			}
			walk(outline.Outlines)
		}
	}
	walk(doc.Body.Outlines)
	return urls, nil
}
