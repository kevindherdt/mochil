package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/reading"
)

// The reading store acknowledges progress in memory and writes it in the
// background, while the KoSync mirror is written immediately. They can
// therefore diverge: this pins what that looks like -- the progress request
// succeeds, the mirror is durable, and the reading failure is reported by the
// store and retried, never answered as a 500 that would make the plugin resend
// a position the server has already accepted.
func TestPluginProgressKeepsTheKoSyncMirrorWhenTheReadingStoreCannotSave(t *testing.T) {
	server, token := browseTestServer(t)
	dir := t.TempDir()
	readingPath := filepath.Join(dir, "reading.json")
	store, err := reading.Open(readingPath)
	if err != nil {
		t.Fatal(err)
	}
	server.reading = store
	server.readingSync = reading.NewSyncCoordinator(store)
	document := strings.Repeat("cd", 16)
	if _, err := server.readingSync.Link("reader", "books", "a1", document); err != nil {
		t.Fatal(err)
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	syncPath := filepath.Join(dir, "kosync.json")
	server.kosync, err = newKoSyncStore(syncPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.kosync.ensureUser("shared", "secret"); err != nil {
		t.Fatal(err)
	}
	server.cfg.KoSyncUsername = "shared"
	if err := os.Remove(readingPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(readingPath, 0700); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, strings.NewReader(`{"progress":"/body/p[1]","percentage":0.4,"timestamp":123}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if position, _ := store.Get("reader", "books", "a1"); position.Fraction != 0.4 {
		t.Fatalf("the acknowledged position was dropped: %+v", position)
	}
	if err := store.Flush(); err == nil {
		t.Fatal("the unreadable reading state was not reported by the store")
	}
	reopened, err := newKoSyncStore(syncPath, false)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.data.Users["shared"].Progress[document]
	if !ok || got.Percentage != 0.4 {
		t.Fatalf("durable mirror = %+v, present %v", got, ok)
	}
}
