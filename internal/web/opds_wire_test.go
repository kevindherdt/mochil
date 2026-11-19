package web

import (
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
)

func TestOPDSWireFeedUsesNewestModificationAndPageNamespace(t *testing.T) {
	older := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	newer := older.Add(24 * time.Hour)
	server := &Server{cfg: Config{BasePath: "/opds", PageStreaming: true}}
	feed := server.feed("Shelf", "urn:test", []catalog.Book{{Modified: newer}, {Modified: older}})

	if feed.Updated != newer.Format(time.RFC3339) || feed.XMLNSPSE == "" {
		t.Fatalf("feed metadata = updated %q pse %q", feed.Updated, feed.XMLNSPSE)
	}
	recorder := httptest.NewRecorder()
	server.writeFeed(recorder, feed, "navigation")
	if got := recorder.Header().Get("Content-Type"); got != atomType+"navigation" {
		t.Fatalf("content type = %q", got)
	}
	var decoded Feed
	if err := xml.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil || decoded.Title != "Shelf" {
		t.Fatalf("decode feed: title=%q err=%v body=%q", decoded.Title, err, recorder.Body.String())
	}
}

func TestOpenSearchEscapesConfiguredCatalogueURL(t *testing.T) {
	server := &Server{cfg: Config{BasePath: "/opds?from=books&kind=all"}}
	recorder := httptest.NewRecorder()
	server.openSearch(recorder, httptest.NewRequest("GET", "/opensearch.xml", nil))

	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "&amp;kind=all/search?q={searchTerms}") {
		t.Fatalf("open search response = %d %q", recorder.Code, recorder.Body.String())
	}
}

// An empty catalogue is a normal first-run state. Dating its feed 1970 tells every reader the
// server is broken and makes clients with an if-modified-since cache keep re-downloading nothing.
func TestEmptyCatalogueFeedIsDatedNow(t *testing.T) {
	stamp := updated(nil)
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		t.Fatalf("updated = %q: %v", stamp, err)
	}
	if parsed.Year() < 2000 {
		t.Fatalf("empty catalogue feed is dated %s", stamp)
	}
	if drift := time.Since(parsed); drift > time.Minute || drift < -time.Minute {
		t.Fatalf("empty catalogue feed dated %s, which is not now (%s)", stamp, drift)
	}
}

// A catalogue with books keeps using the newest modification, not the clock.
func TestCatalogueFeedUsesTheNewestBook(t *testing.T) {
	newest := time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)
	older := newest.Add(-48 * time.Hour)
	if got := updated([]catalog.Book{{Modified: older}, {Modified: newest}}); got != newest.Format(time.RFC3339) {
		t.Fatalf("updated = %q, want %q", got, newest.Format(time.RFC3339))
	}
}
