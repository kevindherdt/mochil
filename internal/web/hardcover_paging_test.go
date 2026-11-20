package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A shelf reaches thousands of books, so the list endpoints page and filter on the server. The panel asks for
// a page, never for everything, and needs an honest total to say how much is left.
func hardcoverPageCall(t *testing.T, server *multiServer, path string) (items []map[string]any, total float64) {
	t.Helper()
	response := httptest.NewRecorder()
	server.apiHardcover(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%s: status=%d body=%s", path, response.Code, response.Body)
	}
	var body struct {
		Items []map[string]any `json:"items"`
		Total float64          `json:"total"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return body.Items, body.Total
}

func TestHardcoverListsPageInsteadOfSendingTheWholeShelf(t *testing.T) {
	server := editorTestServer(t)
	for i := 0; i < 120; i++ {
		server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: fmt.Sprintf("b%03d", i), HardcoverID: "5", HardcoverTitle: "Dune"})
		server.hardcoverSkipped.add(skippedBook{Library: "books", Book: fmt.Sprintf("s%03d", i), Title: fmt.Sprintf("Unmatched %d", i)})
	}

	items, total := hardcoverPageCall(t, server, "/api/metadata/hardcover/matches?limit=25&offset=0")
	if len(items) != 25 || total != 120 {
		t.Fatalf("first page: %d items of %v", len(items), total)
	}
	items, total = hardcoverPageCall(t, server, "/api/metadata/hardcover/matches?limit=25&offset=100")
	if len(items) != 20 || total != 120 {
		t.Fatalf("last page: %d items of %v", len(items), total)
	}

	skippedItems, _ := hardcoverPageCall(t, server, "/api/metadata/hardcover/skipped?limit=10&offset=0")
	if len(skippedItems) != 10 {
		t.Fatalf("skipped page: %d items", len(skippedItems))
	}
}

func TestHardcoverListsFilterByTitleOnTheServer(t *testing.T) {
	server := editorTestServer(t)
	server.hardcoverSkipped.add(skippedBook{Library: "books", Book: "a", Title: "Chapter 1"})
	server.hardcoverSkipped.add(skippedBook{Library: "books", Book: "b", Title: "Neuromancer"})

	items, total := hardcoverPageCall(t, server, "/api/metadata/hardcover/skipped?q=neuro")
	if len(items) != 1 || total != 1 || items[0]["title"] != "Neuromancer" {
		t.Fatalf("filtered: %v items of %v", items, total)
	}
}

// "Check these" and the rest are counted apart, so the panel can say "3 to check" without holding the shelf.
func TestHardcoverMatchesSplitDoubtfulFromCertainOnTheServer(t *testing.T) {
	server := editorTestServer(t)
	server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: "a1", HardcoverID: "5"})
	server.hardcoverMatches.put(hardcoverMatch{Library: "books", Book: "a2", HardcoverID: "6", Doubt: "several books fit"})

	certain, certainTotal := hardcoverPageCall(t, server, "/api/metadata/hardcover/matches?doubt=0")
	doubtful, doubtfulTotal := hardcoverPageCall(t, server, "/api/metadata/hardcover/matches?doubt=1")
	if certainTotal != 1 || len(certain) != 1 {
		t.Fatalf("certain: %d of %v", len(certain), certainTotal)
	}
	if doubtfulTotal != 1 || len(doubtful) != 1 {
		t.Fatalf("doubtful: %d of %v", len(doubtful), doubtfulTotal)
	}
}

// A limit nobody asked for keeps the response small, and an absurd one cannot be used to demand the whole shelf.
func TestHardcoverPageSizeIsCapped(t *testing.T) {
	page := readHardcoverPage(httptest.NewRequest(http.MethodGet, "/x", nil))
	if page.limit != 50 || page.offset != 0 {
		t.Fatalf("default page: %+v", page)
	}
	huge := readHardcoverPage(httptest.NewRequest(http.MethodGet, "/x?limit=100000", nil))
	if huge.limit != 50 {
		t.Fatalf("an oversized limit was accepted: %d", huge.limit)
	}
}
