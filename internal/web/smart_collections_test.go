package web

import (
	"time"

	"bytes"
	"encoding/json"
	"mochil/internal/catalog"
	"mochil/internal/reading"
	"net/http"
	"net/http/httptest"
	"testing"
)

func smartCollectionRequest(server *multiServer, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.apiCollections(response, request)
	return response
}

func TestSmartCollectionHoldsASearchAndStaysCurrent(t *testing.T) {
	server, _ := browseTestServer(t)
	created := smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Herbert","library_id":"books","rule":{"author":"Frank Herbert"}}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	var collection savedCollection
	if err := json.Unmarshal(created.Body.Bytes(), &collection); err != nil || collection.Rule == nil || collection.Rule.Author != "Frank Herbert" {
		t.Fatalf("collection = %+v %v", collection, err)
	}
	books := func() []string {
		response := smartCollectionRequest(server, http.MethodGet, "/api/collections/"+collection.ID+"/books", "")
		var views []bookView
		if err := json.Unmarshal(response.Body.Bytes(), &views); err != nil {
			t.Fatal(err)
		}
		var titles []string
		for _, view := range views {
			titles = append(titles, view.Title)
		}
		return titles
	}
	if got := books(); len(got) != 2 || got[0] != "Dune" || got[1] != "Dune Messiah" {
		t.Fatalf("books = %v (the author's books, in volume order)", got)
	}
	list := smartCollectionRequest(server, http.MethodGet, "/api/collections?library=books", "")
	var summaries []struct {
		Count int             `json:"count"`
		Rule  *collectionRule `json:"rule"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &summaries); err != nil || len(summaries) != 1 || summaries[0].Count != 2 || summaries[0].Rule == nil {
		t.Fatalf("summaries = %+v %v", summaries, err)
	}
}

func TestSmartCollectionRefusesEmptyRulesAndPickedBooks(t *testing.T) {
	server, _ := browseTestServer(t)
	if response := smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Nothing","library_id":"books","rule":{"query":"  "}}`); response.Code != http.StatusBadRequest {
		t.Fatalf("an empty search made a collection that would hold the whole library: %d", response.Code)
	}
	created := smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Gibson","library_id":"books","rule":{"query":"neuromancer"}}`)
	var collection savedCollection
	_ = json.Unmarshal(created.Body.Bytes(), &collection)
	if response := smartCollectionRequest(server, http.MethodPost, "/api/collections/"+collection.ID+"/members", `{"type":"book","library_id":"books","book_id":"a1"}`); response.Code != http.StatusConflict {
		t.Fatalf("adding a picked book to a smart collection = %d", response.Code)
	}
	if response := smartCollectionRequest(server, http.MethodGet, "/api/collections/"+collection.ID+"/books", ""); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("Neuromancer")) || bytes.Contains(response.Body.Bytes(), []byte("Dune")) {
		t.Fatalf("books = %s", response.Body.String())
	}
}

func TestSmartCollectionShowsInThePluginToo(t *testing.T) {
	server, _ := browseTestServer(t)
	smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Dune books","library_id":"books","rule":{"series":"Dune"}}`)
	reader := server.readingAccount(httptest.NewRequest(http.MethodGet, "/", nil))
	seen := false
	for _, c := range server.collections.list(reader) {
		if c.Name != "Dune books" {
			continue
		}
		seen = true
		if got := len(server.resolveCollection(reader, c)); got != 2 {
			t.Fatalf("resolved %d books", got)
		}
	}
	if !seen {
		t.Fatal("collection was saved under another account")
	}
}

func TestSmartCollectionRulesFilterByReadingStateFormatTagAndAge(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	book := catalog.Book{Format: "epub", Tags: []string{"Sci-Fi", "Classic"}, Modified: now.AddDate(0, 0, -3)}
	none, started, done := reading.Position{}, reading.Position{Fraction: .3}, reading.Position{Finished: true, Fraction: 1}
	for _, tc := range []struct {
		name     string
		rule     collectionRule
		position reading.Position
		has      bool
		want     bool
	}{
		{"unread with no place", collectionRule{Status: "unread"}, none, false, true},
		{"unread but started", collectionRule{Status: "unread"}, started, true, false},
		{"reading", collectionRule{Status: "reading"}, started, true, true},
		{"finished is not reading", collectionRule{Status: "reading"}, done, true, false},
		{"finished", collectionRule{Status: "finished"}, done, true, true},
		{"format matches ignoring case", collectionRule{Format: "EPUB"}, none, false, true},
		{"other format", collectionRule{Format: "pdf"}, none, false, false},
		{"tag", collectionRule{Tag: "classic"}, none, false, true},
		{"missing tag", collectionRule{Tag: "romance"}, none, false, false},
		{"added within a week", collectionRule{AddedDays: 7}, none, false, true},
		{"older than two days", collectionRule{AddedDays: 2}, none, false, false},
		{"everything together", collectionRule{Status: "unread", Format: "epub", Tag: "sci-fi", AddedDays: 7}, none, false, true},
	} {
		if got := tc.rule.matchesDetails(book, tc.position, tc.has, now); got != tc.want {
			t.Errorf("%s: got %v", tc.name, got)
		}
	}
	if (collectionRule{Status: "sleeping"}).valid() || (collectionRule{AddedDays: 99999}).valid() || (collectionRule{}).valid() || !(collectionRule{Status: "unread"}).valid() {
		t.Fatal("a rule needs something to filter on, a known state and a sane age")
	}
}

func TestSmartCollectionByReadingStateUsesTheOwnersProgress(t *testing.T) {
	server, _ := browseTestServer(t)
	smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Unread Dune","library_id":"books","rule":{"series":"Dune","status":"unread"}}`)
	reader := server.readingAccount(httptest.NewRequest(http.MethodGet, "/", nil))
	collection := server.collections.list(reader)[0]
	if got := len(server.resolveCollection(reader, collection)); got != 2 {
		t.Fatalf("nothing is read yet, got %d", got)
	}
	books := server.resolveCollection(reader, collection)
	if _, err := server.reading.Put(reader, reading.Position{LibraryID: "books", BookID: books[0].book.ID, Fraction: .5}, 0); err != nil {
		t.Fatal(err)
	}
	if got := len(server.resolveCollection(reader, collection)); got != 1 {
		t.Fatalf("a started book leaves the unread shelf, got %d", got)
	}
	if got := len(server.resolveCollection("someone-else", collection)); got != 2 {
		t.Fatalf("another account's progress must not count, got %d", got)
	}
}

func TestGapSeriesFindsSkippedWholeVolumesOnly(t *testing.T) {
	number := func(value float64) *float64 { return &value }
	volume := func(series string, index *float64, role catalog.PublicationRole) catalogItem {
		return catalogItem{LibraryID: "books", Book: catalog.Book{Series: series, SeriesIndex: index, PublicationRole: role}}
	}
	items := []catalogItem{
		volume("Skips", number(1), ""), volume("Skips", number(2), ""), volume("Skips", number(4), ""),
		volume("Whole", number(1), ""), volume("Whole", number(2), ""), volume("Whole", number(3), ""),
		volume("Starts late", number(2), ""), volume("Starts late", number(3), ""),
		volume("Lonely", number(5), ""),
		volume("Halves", number(1), ""), volume("Halves", number(1.5), ""), volume("Halves", number(2), ""),
		volume("Parts", number(1), ""), volume("Parts", number(2), ""), volume("Parts", number(3.1), ""), volume("Parts", number(3.2), ""), volume("Parts", number(4), ""),
		volume("Parts and hole", number(1), ""), volume("Parts and hole", number(2.1), ""), volume("Parts and hole", number(2.2), ""), volume("Parts and hole", number(4), ""),
		volume("Extras", number(1), ""), volume("Extras", number(2), ""), volume("Extras", number(7), catalog.PublicationSideStory),
		volume("Extra at 99.9", number(1), ""), volume("Extra at 99.9", number(2), ""), volume("Extra at 99.9", number(99.9), ""),
		volume("By year", number(2019), ""), volume("By year", number(2020), ""),
		volume("", number(1), ""), volume("", number(3), ""),
	}
	gaps := gapSeries(items)
	for series, want := range map[string]bool{"Skips": true, "Starts late": true, "Whole": false, "Lonely": false, "Halves": false, "Parts": false, "Extra at 99.9": false, "Parts and hole": true, "Extras": false, "By year": false, "": false} {
		if got := gaps[gapKey("books", series)]; got != want {
			t.Errorf("%q: got %v, want %v", series, got, want)
		}
	}
	if !(collectionRule{MissingVolumes: true}).valid() {
		t.Fatal("a rule can be only about missing volumes")
	}
}

func TestSmartCollectionSearchCanBeChangedAndOnlyForSmartOnes(t *testing.T) {
	server, _ := browseTestServer(t)
	var smart, plain savedCollection
	_ = json.Unmarshal(smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Herbert","library_id":"books","rule":{"author":"Frank Herbert"}}`).Body.Bytes(), &smart)
	_ = json.Unmarshal(smartCollectionRequest(server, http.MethodPost, "/api/collections", `{"name":"Mine","library_id":"books"}`).Body.Bytes(), &plain)
	count := func() int {
		var views []bookView
		_ = json.Unmarshal(smartCollectionRequest(server, http.MethodGet, "/api/collections/"+smart.ID+"/books", "").Body.Bytes(), &views)
		return len(views)
	}
	if count() != 2 {
		t.Fatalf("starts with %d books", count())
	}
	if response := smartCollectionRequest(server, http.MethodPatch, "/api/collections/"+smart.ID, `{"rule":{"query":"messiah"}}`); response.Code != http.StatusOK {
		t.Fatalf("change search: %d %s", response.Code, response.Body.String())
	}
	if count() != 1 {
		t.Fatalf("after the search changed there are %d books, want 1 (the old author filter is gone)", count())
	}
	if response := smartCollectionRequest(server, http.MethodPatch, "/api/collections/"+smart.ID, `{"name":"Messiah","rule":{"query":"dune"}}`); response.Code != http.StatusOK {
		t.Fatalf("rename and change together: %d", response.Code)
	}
	if got := server.collections.list(server.readingAccount(httptest.NewRequest(http.MethodGet, "/", nil)))[0]; got.Name != "Messiah" || got.Rule.Query != "dune" {
		t.Fatalf("saved = %+v", got)
	}
	for body, want := range map[string]int{`{"rule":{}}`: http.StatusBadRequest, `{"rule":{"status":"nonsense","query":"x"}}`: http.StatusBadRequest, `{}`: http.StatusBadRequest} {
		if response := smartCollectionRequest(server, http.MethodPatch, "/api/collections/"+smart.ID, body); response.Code != want {
			t.Fatalf("%s: %d, want %d", body, response.Code, want)
		}
	}
	if response := smartCollectionRequest(server, http.MethodPatch, "/api/collections/"+plain.ID, `{"rule":{"query":"x"}}`); response.Code != http.StatusBadRequest {
		t.Fatalf("a collection of picked books has no search to change: %d", response.Code)
	}
	if response := smartCollectionRequest(server, http.MethodPatch, "/api/collections/"+plain.ID, `{"name":"Renamed"}`); response.Code != http.StatusOK {
		t.Fatalf("rename still works: %d", response.Code)
	}
}
