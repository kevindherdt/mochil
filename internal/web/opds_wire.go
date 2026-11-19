package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mochil/internal/catalog"
)

const (
	atomType       = "application/atom+xml;profile=opds-catalog;kind="
	navRel         = "subsection"
	acquisitionRel = "http://opds-spec.org/acquisition"
)

// Feed and its child types are the shared Atom/OPDS wire contract used by
// single- and multi-library catalogue routes.
type Feed struct {
	XMLName  xml.Name `xml:"feed"`
	XMLNS    string   `xml:"xmlns,attr"`
	XMLNSPSE string   `xml:"xmlns:pse,attr,omitempty"`
	ID       string   `xml:"id"`
	Title    string   `xml:"title"`
	Updated  string   `xml:"updated"`
	Links    []Link   `xml:"link"`
	Entries  []Entry  `xml:"entry"`
}

type Link struct {
	Rel      string `xml:"rel,attr,omitempty"`
	Href     string `xml:"href,attr"`
	Type     string `xml:"type,attr,omitempty"`
	Title    string `xml:"title,attr,omitempty"`
	PSECount int    `xml:"pse:count,attr,omitempty"`
}

type Entry struct {
	ID      string   `xml:"id"`
	Title   string   `xml:"title"`
	Updated string   `xml:"updated"`
	Authors []Person `xml:"author,omitempty"`
	Content Content  `xml:"content"`
	Links   []Link   `xml:"link"`
}

type Person struct {
	Name string `xml:"name"`
}

type Content struct {
	Type string `xml:"type,attr"`
	Text string `xml:",chardata"`
}

func (s *Server) openSearch(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/opensearchdescription+xml")
	template := s.catalogHref("/search") + "?q={searchTerms}"
	_, _ = fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?><OpenSearchDescription xmlns=\"http://a9.com/-/spec/opensearch/1.1/\"><ShortName>Mochil</ShortName><Description>Search the ebook library</Description><Url type=\"application/atom+xml;profile=opds-catalog;kind=acquisition\" template=\"%s\"/></OpenSearchDescription>", xmlEscape(template))
}

func (s *Server) feed(title, id string, books []catalog.Book) Feed {
	return s.feedUpdated(title, id, updated(books))
}

// feedUpdated builds a feed from a timestamp the caller already resolved, for a
// handler that has the catalogue's newest modification without its books.
func (s *Server) feedUpdated(title, id, when string) Feed {
	feed := Feed{XMLNS: "http://www.w3.org/2005/Atom", ID: id, Title: title, Updated: when, Links: []Link{{Rel: "start", Href: s.catalogHref(""), Type: atomType + "navigation"}}}
	if s.cfg.PageStreaming {
		feed.XMLNSPSE = "http://vaemendis.net/opds-pse/ns"
	}
	return feed
}

func (s *Server) writeFeed(w http.ResponseWriter, feed Feed, kind string) {
	w.Header().Set("Content-Type", atomType+kind)
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(feed)
}

func navEntry(title, description, href, when, key string) Entry {
	return Entry{ID: "urn:opdshelf:nav:" + hash(key), Title: title, Updated: when, Content: Content{Type: "text", Text: description}, Links: []Link{{Rel: navRel, Href: href, Type: atomType + "navigation"}}}
}

func updated(books []catalog.Book) string {
	var latest time.Time
	for _, book := range books {
		if book.Modified.After(latest) {
			latest = book.Modified
		}
	}
	return updatedAt(latest)
}

// updatedAt formats a feed timestamp. An empty catalogue is a first-run state,
// not a 1970 archive: readers treat an epoch-dated feed as a broken server, and
// anything caching by date would keep asking for it.
func updatedAt(latest time.Time) string {
	if latest.IsZero() {
		latest = time.Now()
	}
	return latest.UTC().Format(time.RFC3339)
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func xmlEscape(value string) string {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}
