package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/externallibrary"
)

func TestExternalLibraryMembershipAndProgressAPI(t *testing.T) {
	store, err := externallibrary.Open(filepath.Join(t.TempDir(), "external.json"))
	if err != nil {
		t.Fatal(err)
	}
	server := &multiServer{externalLibrary: store}
	add := httptest.NewRequest(http.MethodPost, "/api/external-library", strings.NewReader(`{"provider_id":"opds","source_id":"catalogue","remote_title_id":"urn:book:one","name":"One","execution_location":"device"}`))
	add.Header.Set("Content-Type", "application/json")
	added := httptest.NewRecorder()
	server.apiExternalLibrary(added, add)
	if added.Code != http.StatusCreated || !strings.Contains(added.Body.String(), `"name":"One"`) {
		t.Fatalf("add=%d %s", added.Code, added.Body.String())
	}
	id := externallibrary.StableID("opds", "catalogue", "urn:book:one")
	put := httptest.NewRequest(http.MethodPut, "/api/external-library/"+id+"/progress", strings.NewReader(`{"base_revision":0,"progress":{"remote_chapter_id":"chapter-1","locator":"page:8","fraction":0.5,"device_id":"phone","device_name":"Samsung Galaxy"}}`))
	put.Header.Set("Content-Type", "application/json")
	progress := httptest.NewRecorder()
	server.apiExternalLibrary(progress, put)
	if progress.Code != http.StatusOK || !strings.Contains(progress.Body.String(), `"revision":1`) || !strings.Contains(progress.Body.String(), `"device_name":"Samsung Galaxy"`) {
		t.Fatalf("progress=%d %s", progress.Code, progress.Body.String())
	}
	conflictRequest := httptest.NewRequest(http.MethodPut, "/api/external-library/"+id+"/progress", strings.NewReader(`{"base_revision":0,"progress":{"remote_chapter_id":"chapter-1","locator":"page:2","fraction":0.1,"device_id":"tablet"}}`))
	conflictRequest.Header.Set("Content-Type", "application/json")
	conflict := httptest.NewRecorder()
	server.apiExternalLibrary(conflict, conflictRequest)
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), `"locator":"page:8"`) || !strings.Contains(conflict.Body.String(), `"revision":1`) {
		t.Fatalf("conflict=%d %s", conflict.Code, conflict.Body.String())
	}
	list := httptest.NewRecorder()
	server.apiExternalLibrary(list, httptest.NewRequest(http.MethodGet, "/api/external-library", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"execution_location":"device"`) {
		t.Fatalf("list=%d %s", list.Code, list.Body.String())
	}
}

func TestExternalLibraryProgressAPIAcknowledgesSameReadiumPassage(t *testing.T) {
	store, _ := externallibrary.Open("")
	title, _ := store.Put("default", externallibrary.Title{ProviderID: "opds", SourceID: "catalogue", RemoteTitleID: "urn:book:one", Name: "One", ExecutionLocation: externallibrary.ExecutionDevice})
	server := &multiServer{externalLibrary: store}
	put := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPut, "/api/external-library/"+title.ID+"/progress", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		server.apiExternalLibrary(response, request)
		return response
	}
	first := put(`{"base_revision":0,"progress":{"remote_chapter_id":"publication","locator":"{\"href\":\"chapter.xhtml\",\"text\":{\"before\":\"before\",\"highlight\":\"same words\",\"after\":\"after\"}}","fraction":0.31,"device_id":"tablet"}}`)
	if first.Code != http.StatusOK {
		t.Fatalf("initial=%d %s", first.Code, first.Body.String())
	}
	stale := put(`{"base_revision":0,"progress":{"remote_chapter_id":"publication","locator":"{\"href\":\"chapter.xhtml\",\"text\":{\"before\":\"before\",\"highlight\":\"same words\",\"after\":\"after\"},\"locations\":{\"totalProgression\":0.46}}","fraction":0.46,"device_id":"phone"}}`)
	if stale.Code != http.StatusOK || !strings.Contains(stale.Body.String(), `"fraction":0.31`) || !strings.Contains(stale.Body.String(), `"device_id":"tablet"`) {
		t.Fatalf("stale acknowledgement=%d %s", stale.Code, stale.Body.String())
	}
}

func TestExternalLibraryAPIRejectsRetiredExecutableProviders(t *testing.T) {
	store, err := externallibrary.Open(filepath.Join(t.TempDir(), "external.json"))
	if err != nil {
		t.Fatal(err)
	}
	server := &multiServer{externalLibrary: store}
	request := httptest.NewRequest(http.MethodPost, "/api/external-library", strings.NewReader(`{"provider_id":"plugin","source_id":"42","remote_title_id":"one","name":"One","execution_location":"server"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.apiExternalLibrary(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Only device-read OPDS") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestExternalLibraryAPIHidesRetiredProviderState(t *testing.T) {
	store, _ := externallibrary.Open("")
	legacy, err := store.Put("reader", externallibrary.Title{ProviderID: "legacy", SourceID: "source", RemoteTitleID: "one", Name: "Legacy", ExecutionLocation: externallibrary.ExecutionServer})
	if err != nil {
		t.Fatal(err)
	}
	server := &multiServer{externalLibrary: store}
	list := httptest.NewRecorder()
	server.apiExternalLibrary(list, httptest.NewRequest(http.MethodGet, "/api/external-library", nil))
	if strings.Contains(list.Body.String(), "Legacy") {
		t.Fatalf("retired provider leaked: %s", list.Body.String())
	}
	progress := httptest.NewRecorder()
	server.apiExternalLibrary(progress, httptest.NewRequest(http.MethodGet, "/api/external-library/"+legacy.ID+"/progress", nil))
	if progress.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", progress.Code, progress.Body.String())
	}
}

func TestAndroidExternalLibraryRequiresDeviceToken(t *testing.T) {
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")}, []Library{{ID: "books", Name: "Books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/android/v1/external-library", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
