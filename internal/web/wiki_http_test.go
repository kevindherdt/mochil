package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/wiki"
)

func TestWikiAPIConflictTrashHistoryAndExport(t *testing.T) {
	store, err := wiki.Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{cfg: MultiConfig{Username: "alice"}, wiki: store}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		w := httptest.NewRecorder()
		m.wikiAPI(w, r)
		return w
	}
	created := request(http.MethodPost, "/api/wiki", `{"title":"Home","content":"Hello"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var page wiki.Page
	if err := json.Unmarshal(created.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	createdFolder := request(http.MethodPost, "/api/wiki/folders", `{"folder":"Projects/Books"}`)
	if createdFolder.Code != http.StatusCreated {
		t.Fatalf("create folder: %d %s", createdFolder.Code, createdFolder.Body.String())
	}
	folders := request(http.MethodGet, "/api/wiki/folders", "")
	if folders.Code != http.StatusOK || !strings.Contains(folders.Body.String(), "Projects/Books") {
		t.Fatalf("list folders: %d %s", folders.Code, folders.Body.String())
	}
	if response := request(http.MethodPost, "/api/wiki", `{"title":"Other","content":"Hello"} {}`); response.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON status = %d, want 400", response.Code)
	}
	oversized := `{"title":"Other","content":"` + strings.Repeat("x", (2<<20)+4096) + `"}`
	if response := request(http.MethodPost, "/api/wiki", oversized); response.Code != http.StatusBadRequest {
		t.Fatalf("oversized JSON status = %d, want 400", response.Code)
	}
	conflict := request(http.MethodPut, "/api/wiki/"+page.ID, `{"title":"Home","content":"Changed","expected_revision":9}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict: %d", conflict.Code)
	}
	updated := request(http.MethodPut, "/api/wiki/"+page.ID, `{"title":"Home","content":"Changed","expected_revision":1}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update: %d", updated.Code)
	}
	history := request(http.MethodGet, "/api/wiki/"+page.ID+"/history", "")
	if history.Code != http.StatusOK || !strings.Contains(history.Body.String(), "Hello") {
		t.Fatalf("history: %d %s", history.Code, history.Body.String())
	}
	deleted := request(http.MethodDelete, "/api/wiki/"+page.ID+"?expected_revision=2", "")
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete: %d", deleted.Code)
	}
	trash := request(http.MethodGet, "/api/wiki?deleted=1", "")
	if !strings.Contains(trash.Body.String(), page.ID) {
		t.Fatalf("trash: %s", trash.Body.String())
	}
	wrongPurge := request(http.MethodPost, "/api/wiki/"+page.ID+"/purge", `{"expected_revision":3,"confirmation":"PURGE Wrong"}`)
	if wrongPurge.Code != http.StatusBadRequest {
		t.Fatalf("wrong purge confirmation: %d %s", wrongPurge.Code, wrongPurge.Body.String())
	}
	earlyPurge := request(http.MethodPost, "/api/wiki/"+page.ID+"/purge", `{"expected_revision":3,"confirmation":"PURGE Home"}`)
	if earlyPurge.Code != http.StatusConflict || !strings.Contains(earlyPurge.Body.String(), "7 days") {
		t.Fatalf("early purge: %d %s", earlyPurge.Code, earlyPurge.Body.String())
	}
	exported := request(http.MethodGet, "/api/wiki/export", "")
	if exported.Code != http.StatusOK || !strings.Contains(exported.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("export: %d", exported.Code)
	}
}
