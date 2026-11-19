package podcast

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExportOPMLListsEverySubscribedFeed(t *testing.T) {
	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<rss><channel><title>Feed A</title><item><title>Ep</title><guid>1</guid></item></channel></rss>`))
	}))
	defer feedServer.Close()

	store, err := Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Subscribe(context.Background(), feedServer.URL); err != nil {
		t.Fatal(err)
	}

	out := store.ExportOPML()
	var doc opmlDocument
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("exported OPML did not parse: %v", err)
	}
	if len(doc.Body.Outlines) != 1 {
		t.Fatalf("want 1 outline, got %d", len(doc.Body.Outlines))
	}
	outline := doc.Body.Outlines[0]
	if outline.Type != "rss" {
		t.Errorf("want type=rss, got %q", outline.Type)
	}
	if outline.XMLURL != feedServer.URL {
		t.Errorf("want xmlUrl %q, got %q", feedServer.URL, outline.XMLURL)
	}
	if outline.Title != "Feed A" {
		t.Errorf("want title %q, got %q", "Feed A", outline.Title)
	}
}

func TestExportOPMLWithNoSubscriptionsIsStillValidOPML(t *testing.T) {
	store, err := Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	out := store.ExportOPML()
	var doc opmlDocument
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("exported OPML did not parse: %v", err)
	}
	if len(doc.Body.Outlines) != 0 {
		t.Errorf("want 0 outlines, got %d", len(doc.Body.Outlines))
	}
}

func TestFeedURLsFromOPMLFlattensNestedFolders(t *testing.T) {
	opml := `<?xml version="1.0"?>
<opml version="2.0"><head><title>x</title></head><body>
  <outline text="Tech" title="Tech">
    <outline text="Show A" type="rss" xmlUrl="https://a.example/feed.xml"/>
    <outline text="Show B" type="rss" xmlUrl="https://b.example/feed.xml"/>
  </outline>
  <outline text="Show C" type="rss" xmlUrl="https://c.example/feed.xml"/>
  <outline text="Show A again" type="rss" xmlUrl="https://a.example/feed.xml"/>
</body></opml>`
	urls, err := feedURLsFromOPML([]byte(opml))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://a.example/feed.xml", "https://b.example/feed.xml", "https://c.example/feed.xml"}
	if strings.Join(urls, ",") != strings.Join(want, ",") {
		t.Errorf("want %v, got %v", want, urls)
	}
}

func TestFeedURLsFromOPMLRejectsGarbage(t *testing.T) {
	if _, err := feedURLsFromOPML([]byte("not xml at all")); err == nil {
		t.Error("want an error for non-XML input")
	}
}

func TestImportOPMLReportsEachFeedIndependently(t *testing.T) {
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<rss><channel><title>Good show</title></channel></rss>`))
	}))
	defer goodServer.Close()
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer badServer.Close()

	store, err := Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	opml := `<?xml version="1.0"?><opml version="2.0"><head><title>x</title></head><body>
  <outline text="Good" type="rss" xmlUrl="` + goodServer.URL + `"/>
  <outline text="Bad" type="rss" xmlUrl="` + badServer.URL + `"/>
</body></opml>`

	result, err := store.ImportOPML(context.Background(), []byte(opml))
	if err != nil {
		t.Fatalf("ImportOPML returned an error for a valid document: %v", err)
	}
	if result.Subscribed != 1 {
		t.Errorf("want 1 subscribed, got %d", result.Subscribed)
	}
	if len(result.Failed) != 1 || result.Failed[0].URL != badServer.URL {
		t.Errorf("want exactly the bad feed reported as failed, got %+v", result.Failed)
	}
	if len(store.Feeds()) != 1 {
		t.Errorf("want 1 feed actually subscribed, got %d", len(store.Feeds()))
	}

	// Importing the same document again must not duplicate the good feed.
	result2, err := store.ImportOPML(context.Background(), []byte(opml))
	if err != nil {
		t.Fatal(err)
	}
	if result2.Already != 1 {
		t.Errorf("want 1 already-subscribed on re-import, got %d", result2.Already)
	}
	if len(store.Feeds()) != 1 {
		t.Errorf("want still exactly 1 feed after re-import, got %d", len(store.Feeds()))
	}
}

func TestImportOPMLRejectsNonOPMLInput(t *testing.T) {
	store, err := Open(t.TempDir()+"/podcasts.json", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportOPML(context.Background(), []byte("garbage")); err == nil {
		t.Error("want an error for non-OPML input")
	}
}
