package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKoReaderPluginCanMarkABookFinishedBeforeDownloading(t *testing.T) {
	server, token := browseTestServer(t)
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/finished/books/a1", strings.NewReader(`{"finished":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("mark finished = %d %s", response.Code, response.Body.String())
	}
	position, exists := server.reading.Get("reader", "books", "a1")
	if !exists || !position.Finished {
		t.Fatalf("the book was not marked finished: %+v exists=%v", position, exists)
	}
}

func TestKoReaderPluginCanUnmarkABookFinished(t *testing.T) {
	server, token := browseTestServer(t)
	if _, err := server.reading.SetFinished("reader", "books", "a1", true); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/finished/books/a1", strings.NewReader(`{"finished":false}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unmark finished = %d %s", response.Code, response.Body.String())
	}
	position, exists := server.reading.Get("reader", "books", "a1")
	if !exists || position.Finished {
		t.Fatalf("the book is still marked finished: %+v", position)
	}
}

func TestKoReaderPluginRefusesToMarkAnUnknownBookFinished(t *testing.T) {
	server, token := browseTestServer(t)
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/finished/books/does-not-exist", strings.NewReader(`{"finished":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("mark finished for unknown book = %d %s, want 404", response.Code, response.Body.String())
	}
}

func TestKoReaderPluginFinishedRouteRejectsOtherMethods(t *testing.T) {
	server, token := browseTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/koreader/v1/finished/books/a1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET on finished = %d %s, want 405", response.Code, response.Body.String())
	}
}
