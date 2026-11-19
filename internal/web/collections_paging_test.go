package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

// collectionPagingServer serves one 1,300-book series collection, so a detail
// request is the shape that used to render every card in one write.
func collectionPagingServer(t *testing.T) (http.Handler, savedCollection, []string) {
	t.Helper()
	const total = 1300
	books := make([]catalog.Book, 0, total)
	titles := make([]string, 0, total)
	for index := 1; index <= total; index++ {
		title := fmt.Sprintf("Saga %04d", index)
		titles = append(titles, title)
		books = append(books, catalog.Book{
			ID:          fmt.Sprintf("b%04d", index),
			Path:        fmt.Sprintf("Saga/%04d.epub", index),
			Title:       title,
			Authors:     []string{"Ada Lovelace"},
			Series:      "Saga",
			SeriesIndex: ptr(float64(index)),
			Size:        10,
		})
	}
	handler, err := NewMulti(
		MultiConfig{AllowOpen: true, CollectionsState: filepath.Join(t.TempDir(), "collections.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{books: books}}},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	post := func(target, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	created := post("/api/collections", `{"name":"Saga","library_id":"books"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create collection = %d %s", created.Code, created.Body.String())
	}
	var collection savedCollection
	if err := json.Unmarshal(created.Body.Bytes(), &collection); err != nil {
		t.Fatal(err)
	}
	if member := post("/api/collections/"+collection.ID+"/members", `{"type":"series","library_id":"books","series":"Saga"}`); member.Code != http.StatusNoContent {
		t.Fatalf("add series member = %d %s", member.Code, member.Body.String())
	}
	return handler, collection, titles
}

func getCollectionBooks(t *testing.T, handler http.Handler, id, query string) ([]bookView, int, *httptest.ResponseRecorder) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/collections/"+id+"/books"+query, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %s", query, response.Code, response.Body.String())
	}
	var views []bookView
	if err := json.Unmarshal(response.Body.Bytes(), &views); err != nil {
		t.Fatalf("GET %s body %s: %v", query, response.Body.String(), err)
	}
	total := -1
	if header := response.Header().Get("X-Total-Count"); header != "" {
		fmt.Sscan(header, &total)
	}
	return views, total, response
}

func TestCollectionDetailPagesWithoutChangingTheSingleShotCall(t *testing.T) {
	handler, collection, titles := collectionPagingServer(t)

	// No limit: exactly what the endpoint returned before paging existed.
	all, total, _ := getCollectionBooks(t, handler, collection.ID, "")
	if len(all) != len(titles) || total != len(titles) {
		t.Fatalf("legacy call = %d books, total %d, want %d", len(all), total, len(titles))
	}
	for index, view := range all {
		if view.Title != titles[index] {
			t.Fatalf("legacy order at %d = %q, want %q", index, view.Title, titles[index])
		}
	}
	// An offset alone does not page: the legacy shape stays the legacy shape.
	if withOffset, _, _ := getCollectionBooks(t, handler, collection.ID, "?offset=100"); len(withOffset) != len(titles) {
		t.Fatalf("offset without limit returned %d books, want the whole collection", len(withOffset))
	}

	first, total, _ := getCollectionBooks(t, handler, collection.ID, "?limit=120")
	if len(first) != 120 || total != len(titles) || first[0].Title != titles[0] || first[119].Title != titles[119] {
		t.Fatalf("first page = %d books (total %d), first %q last %q", len(first), total, first[0].Title, first[len(first)-1].Title)
	}
	second, _, _ := getCollectionBooks(t, handler, collection.ID, "?limit=120&offset=120")
	if len(second) != 120 || second[0].Title != titles[120] {
		t.Fatalf("second page starts at %q, want %q", second[0].Title, titles[120])
	}

	// A limit past the cap is clamped, an unusable one falls back to the
	// default page size rather than to the whole collection.
	if capped, _, _ := getCollectionBooks(t, handler, collection.ID, "?limit=5000"); len(capped) != maximumCollectionPageSize {
		t.Fatalf("limit above the cap returned %d books, want %d", len(capped), maximumCollectionPageSize)
	}
	for _, query := range []string{"?limit=abc", "?limit=0", "?limit=-5"} {
		if page, _, _ := getCollectionBooks(t, handler, collection.ID, query); len(page) != defaultCollectionPageSize {
			t.Fatalf("%s returned %d books, want the %d-book default", query, len(page), defaultCollectionPageSize)
		}
	}

	// The tail, and an offset past the end, are windows over the same total.
	last, _, _ := getCollectionBooks(t, handler, collection.ID, "?limit=120&offset=1290")
	if len(last) != 10 || last[9].Title != titles[len(titles)-1] {
		t.Fatalf("tail = %d books, last %q", len(last), last[len(last)-1].Title)
	}
	empty, total, response := getCollectionBooks(t, handler, collection.ID, "?limit=120&offset=5000")
	if len(empty) != 0 || total != len(titles) || response.Body.String() != "[]\n" {
		t.Fatalf("past the end = %d books, total %d, body %q", len(empty), total, response.Body.String())
	}
}

func TestCollectionDetailPagesWalkTheSameBooksAsTheSingleShot(t *testing.T) {
	handler, collection, titles := collectionPagingServer(t)
	walked := make([]string, 0, len(titles))
	for offset := 0; offset < len(titles); offset += defaultCollectionPageSize {
		page, total, _ := getCollectionBooks(t, handler, collection.ID, fmt.Sprintf("?limit=%d&offset=%d", defaultCollectionPageSize, offset))
		if total != len(titles) {
			t.Fatalf("page at %d reported total %d, want %d", offset, total, len(titles))
		}
		for _, view := range page {
			walked = append(walked, view.Title)
		}
	}
	if len(walked) != len(titles) {
		t.Fatalf("pages walked %d books, want %d", len(walked), len(titles))
	}
	for index, title := range titles {
		if walked[index] != title {
			t.Fatalf("walk at %d = %q, want %q", index, walked[index], title)
		}
	}
}
