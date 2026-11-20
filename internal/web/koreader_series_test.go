package web

import (
	"encoding/json"
	"net/http"
	"testing"
)

func seriesPage(t *testing.T, server *multiServer, token, path string) koReaderSeriesPage {
	t.Helper()
	response := pluginUpdateRequest(server, http.MethodGet, path, token)
	if response.Code != http.StatusOK {
		t.Fatalf("%s = %d %s", path, response.Code, response.Body.String())
	}
	var page koReaderSeriesPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return page
}

func TestKoReaderSeriesOffersTheSurroundingVolumes(t *testing.T) {
	server, token := browseTestServer(t)

	first := seriesPage(t, server, token, "/api/koreader/v1/series/books/a1")
	if first.Series != "Dune" || first.Index == nil || *first.Index != 1 {
		t.Fatalf("series = %q index = %v", first.Series, first.Index)
	}
	if first.Previous != nil {
		t.Fatalf("volume 1 has nothing before it, got %+v", first.Previous)
	}
	if first.Next == nil || first.Next.ID != "a2" {
		t.Fatalf("next = %+v, want Dune Messiah", first.Next)
	}

	second := seriesPage(t, server, token, "/api/koreader/v1/series/books/a2")
	if second.Previous == nil || second.Previous.ID != "a1" {
		t.Fatalf("previous = %+v, want Dune", second.Previous)
	}
	if second.Next != nil {
		t.Fatalf("volume 2 is the last one here, got %+v", second.Next)
	}
}

// A standalone book is a valid question with an empty answer: the plugin asks only for books that
// carry a series, so reaching this route without one means the catalogue changed under the dialog.
func TestKoReaderSeriesOutsideASeriesHasNoNeighbours(t *testing.T) {
	server, token := browseTestServer(t)
	page := seriesPage(t, server, token, "/api/koreader/v1/series/books/n1")
	if page.Series != "" || page.Previous != nil || page.Next != nil {
		t.Fatalf("standalone book answered with %+v", page)
	}
}

func TestKoReaderSeriesIsDeviceGatedAndKnowsMissingBooks(t *testing.T) {
	server, token := browseTestServer(t)
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/series/books/a1", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("without a token = %d, want 401", response.Code)
	}
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/series/books/missing", token); response.Code != http.StatusNotFound {
		t.Fatalf("unknown book = %d, want 404", response.Code)
	}
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/series/nolibrary/a1", token); response.Code != http.StatusNotFound {
		t.Fatalf("unknown library = %d, want 404", response.Code)
	}
}
