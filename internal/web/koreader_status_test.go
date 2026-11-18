package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mochil/internal/reading"
)

func statusRequest(server *multiServer, method, path, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	return response
}

func seedReading(t *testing.T, server *multiServer) {
	t.Helper()
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .4}, 0); err != nil {
		t.Fatal(err)
	}
}

// A device has to be able to park a book itself; otherwise the status only the browser can set
// silently decides what the device's continue shelf shows.
func TestKoReaderStatusParksAndUnparksABook(t *testing.T) {
	server, token := browseTestServer(t)
	seedReading(t, server)

	if got := statusRequest(server, http.MethodPut, "/api/koreader/v1/status/books/a1", "", `{"status":"on_hold"}`); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", got.Code)
	}
	if got := statusRequest(server, http.MethodPost, "/api/koreader/v1/status/books/a1", token, `{"status":"on_hold"}`); got.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", got.Code)
	}
	if got := statusRequest(server, http.MethodPut, "/api/koreader/v1/status/books/a1", token, `{"status":"sideways"}`); got.Code != http.StatusBadRequest {
		t.Fatalf("unknown status = %d, want 400", got.Code)
	}

	parked := statusRequest(server, http.MethodPut, "/api/koreader/v1/status/books/a1", token, `{"status":"on_hold"}`)
	if parked.Code != http.StatusOK {
		t.Fatalf("park = %d %s", parked.Code, parked.Body.String())
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(parked.Body.Bytes(), &result); err != nil || result.Status != "on_hold" {
		t.Fatalf("park response = %s", parked.Body.String())
	}
	if entries := continueEntries(t, server, token); len(entries) != 0 {
		t.Fatalf("parked book still on the continue shelf: %v", entries)
	}

	unparked := statusRequest(server, http.MethodPut, "/api/koreader/v1/status/books/a1", token, `{"status":"reading"}`)
	if unparked.Code != http.StatusOK {
		t.Fatalf("unpark = %d %s", unparked.Code, unparked.Body.String())
	}
	if entries := continueEntries(t, server, token); len(entries) != 1 {
		t.Fatalf("continue after unparking = %v, want the book back", entries)
	}
}

func continueEntries(t *testing.T, server *multiServer, token string) []string {
	t.Helper()
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/continue", token)
	var page struct {
		Continue []struct {
			Book *struct {
				ID string `json:"id"`
			} `json:"book"`
		} `json:"continue"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("continue: %d %s", response.Code, response.Body.String())
	}
	ids := make([]string, 0, len(page.Continue))
	for _, entry := range page.Continue {
		if entry.Book != nil {
			ids = append(ids, entry.Book.ID)
		}
	}
	return ids
}

// The device needs to see what it parked; the web and the Android app both have that shelf already.
func TestKoReaderHeldShelfListsOnlyParkedBooks(t *testing.T) {
	server, token := browseTestServer(t)
	seedReading(t, server)
	if _, err := server.reading.Put("reader", reading.Position{LibraryID: "books", BookID: "a2", Fraction: .3}, 0); err != nil {
		t.Fatal(err)
	}
	if got := statusRequest(server, http.MethodPut, "/api/koreader/v1/status/books/a2", token, `{"status":"dropped"}`); got.Code != http.StatusOK {
		t.Fatalf("drop = %d %s", got.Code, got.Body.String())
	}

	if ids := continueEntries(t, server, token); len(ids) != 1 || ids[0] != "a1" {
		t.Fatalf("continue = %v, want only the book still being read", ids)
	}
	if ids := heldEntries(t, server, token); len(ids) != 1 || ids[0] != "a2" {
		t.Fatalf("held = %v, want only the dropped book", ids)
	}
}

func heldEntries(t *testing.T, server *multiServer, token string) []string {
	t.Helper()
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/continue?held=true", token)
	var page struct {
		Continue []struct {
			Book *struct {
				ID string `json:"id"`
			} `json:"book"`
		} `json:"continue"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || response.Code != http.StatusOK {
		t.Fatalf("held: %d %s", response.Code, response.Body.String())
	}
	ids := make([]string, 0, len(page.Continue))
	for _, entry := range page.Continue {
		if entry.Book != nil {
			ids = append(ids, entry.Book.ID)
		}
	}
	return ids
}
