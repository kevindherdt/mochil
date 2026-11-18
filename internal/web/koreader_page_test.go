package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/reading"
)

// koReaderPageTestServer builds a minimal multiServer wired for the koReaderPlugin handler
// (devices, reading store, kosync and PublicURL all required or koReaderPlugin 501s) plus one
// page-streaming-enabled manga library backed by streamingBackend (defined in web_test.go).
func koReaderPageTestServer(t *testing.T, pageStreaming bool) (*multiServer, string) {
	t.Helper()
	directory := t.TempDir()
	devices, err := newKoReaderDeviceStore(filepath.Join(directory, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := reading.Open(filepath.Join(directory, "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := devices.issue("reader")
	if err != nil {
		t.Fatal(err)
	}
	backend := streamingBackend{book: catalog.Book{
		ID: "comic", Path: "Comic.cbz", Title: "Comic", Format: "cbz",
		MediaType: "application/vnd.comicbook+zip", PageCount: 3, PageMediaType: "image/jpeg",
	}}
	server := &multiServer{
		cfg:       MultiConfig{PublicURL: "https://books.example"},
		devices:   devices,
		reading:   store,
		kosync:    &koSyncStore{},
		libraries: []Library{{ID: "manga", Kind: "manga", Store: backend, PageStreaming: pageStreaming}},
	}
	return server, token
}

func koReaderPageRequest(server *multiServer, path, credential string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	return response
}

func TestKoReaderPageRequiresDeviceTokenAndServesPageBytes(t *testing.T) {
	server, token := koReaderPageTestServer(t, true)
	path := "/api/koreader/v1/pages/manga/comic/1"

	if response := koReaderPageRequest(server, path, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d", response.Code)
	}
	if response := koReaderPageRequest(server, path, "not-a-real-token"); response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status=%d", response.Code)
	}

	response := koReaderPageRequest(server, path, token)
	if response.Code != http.StatusOK || response.Body.String() != "page" || response.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("authenticated page fetch status=%d type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}

	if outOfRange := koReaderPageRequest(server, "/api/koreader/v1/pages/manga/comic/99", token); outOfRange.Code != http.StatusBadRequest {
		t.Fatalf("out of range page status=%d", outOfRange.Code)
	}
	if missingBook := koReaderPageRequest(server, "/api/koreader/v1/pages/manga/missing/0", token); missingBook.Code != http.StatusNotFound {
		t.Fatalf("missing book status=%d", missingBook.Code)
	}
}

func TestKoReaderPageIsUnavailableWhenPageStreamingIsDisabled(t *testing.T) {
	server, token := koReaderPageTestServer(t, false)
	response := koReaderPageRequest(server, "/api/koreader/v1/pages/manga/comic/1", token)
	if response.Code != http.StatusNotFound {
		t.Fatalf("page streaming disabled status=%d", response.Code)
	}
}
