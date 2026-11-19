package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/reading"
)

func TestKoReaderContinuePagination(t *testing.T) {
	server, token := browseTestServer(t)
	books := make([]catalog.Book, 7)
	positions := map[string]reading.Position{}
	for i := range books {
		id := fmt.Sprintf("b%d", i)
		books[i] = catalog.Book{ID: id, Title: id, Path: id + ".epub"}
		positions["books\x00"+id] = reading.Position{LibraryID: "books", BookID: id, UpdatedAt: 100, Fraction: .5}
	}
	positions["books\x00b6"] = reading.Position{LibraryID: "books", BookID: "b6", UpdatedAt: 150, Fraction: .5}
	positions["books\x00missing"] = reading.Position{LibraryID: "books", BookID: "missing", UpdatedAt: 200, Fraction: .5}
	positions["removed\x00secret"] = reading.Position{LibraryID: "removed", BookID: "secret", UpdatedAt: 300, Fraction: .5}
	positions["books\x00finished"] = reading.Position{LibraryID: "books", BookID: "finished", UpdatedAt: 300, Finished: true}
	positions["books\x00dismissed"] = reading.Position{LibraryID: "books", BookID: "dismissed", UpdatedAt: 300, Dismissed: true}
	state, err := json.Marshal(map[string]any{"version": 1, "accounts": map[string]any{"reader": positions, "other": map[string]reading.Position{"books\x00b0": {LibraryID: "books", BookID: "b0", UpdatedAt: 300, Fraction: .5}}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "reading.json")
	if err := os.WriteFile(path, state, 0600); err != nil {
		t.Fatal(err)
	}
	server.reading, err = reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server.libraries[0].Store = staticBackend{books: books}
	const endpoint = "/api/koreader/v1/continue"
	if got := pluginUpdateRequest(server, http.MethodGet, endpoint, ""); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: %d", got.Code)
	}
	if got := pluginUpdateRequest(server, http.MethodPost, endpoint, token); got.Code != http.StatusNotFound {
		t.Fatalf("POST: %d", got.Code)
	}
	var ids []string
	for offset := 0; offset < 9; offset += 3 {
		response := pluginUpdateRequest(server, http.MethodGet, fmt.Sprintf("%s?limit=3&offset=%d", endpoint, offset), token)
		var page struct {
			Total    int                `json:"total"`
			Continue []koReaderHomeBook `json:"continue"`
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &page) != nil {
			t.Fatalf("page: %d %s", response.Code, response.Body.String())
		}
		if page.Total != 7 || page.Continue == nil {
			t.Fatalf("page: %+v", page)
		}
		for _, item := range page.Continue {
			if item.Book == nil {
				t.Fatal("missing book")
			}
			ids = append(ids, item.Book.ID)
		}
	}
	if want := []string{"b6", "b0", "b1", "b2", "b3", "b4", "b5"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("ids=%v want=%v", ids, want)
	}
	other, err := server.devices.issue("other")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query, token string
		total        int
	}{
		{"?library=books", token, 7},
		{"?library=removed", token, 0},
		{"?offset=9223372036854775807&limit=100", token, 7},
		{"", other, 1},
	} {
		response := pluginUpdateRequest(server, http.MethodGet, endpoint+tc.query, tc.token)
		var page struct {
			Total    int                `json:"total"`
			Continue []koReaderHomeBook `json:"continue"`
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &page) != nil || page.Total != tc.total || page.Continue == nil || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("query %s: %d %s", tc.query, response.Code, response.Body.String())
		}
	}
}
