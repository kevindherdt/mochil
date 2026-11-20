package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/annotations"
	"mochil/internal/catalog"
	"mochil/internal/reading"
)

func transferServer(t *testing.T) (*multiServer, string, string) {
	t.Helper()
	server := editorTestServer(t)
	root := t.TempDir()
	one, two := filepath.Join(root, "one"), filepath.Join(root, "two")
	for rel, body := range map[string]string{"Dune/Dune 1.epub": "d1", "Dune/Dune 2.epub": "d2", "n.epub": "n"} {
		path := filepath.Join(one, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(two, 0o755); err != nil {
		t.Fatal(err)
	}
	server.libraries[0].SourceType, server.libraries[0].SourcePath = "filesystem", one
	server.libraries = append(server.libraries, Library{ID: "shelf", Name: "Shelf", Kind: "books", SourceType: "filesystem", SourcePath: two, Store: server.metadataOverrides.Wrap("shelf", staticBackend{})})
	return server, one, two
}

func present(root, rel string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func TestTransferPreviewChangesNothingAndAMoveCarriesEditsAndReadingAndCanBeUndone(t *testing.T) {
	server, one, two := transferServer(t)
	if err := server.metadataOverrides.Apply("books", map[string]map[string]string{"a1": {"title": "Dune (corrected)"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.reading.Put("default", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .4}, 0); err != nil {
		t.Fatal(err)
	}
	body := `{"library":"books","to":"shelf","ids":["a1","n1"],"mode":"move","preview":true}`
	preview := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", body)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), `"ready":2`) || !present(one, "n.epub") || present(two, "n.epub") {
		t.Fatalf("preview = %d %s", preview.Code, preview.Body.String())
	}
	done := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", strings.Replace(body, `"preview":true`, `"preview":false`, 1))
	var result struct {
		Done   int      `json:"done"`
		Token  string   `json:"token"`
		Rescan []string `json:"rescan"`
	}
	if err := json.Unmarshal(done.Body.Bytes(), &result); err != nil || done.Code != http.StatusOK || result.Done != 2 || result.Token == "" || len(result.Rescan) != 2 {
		t.Fatalf("move = %d %s", done.Code, done.Body.String())
	}
	if present(one, "n.epub") || present(one, "Dune/Dune 1.epub") || !present(two, "n.epub") || !present(two, "Dune/Dune 1.epub") || !present(one, "Dune/Dune 2.epub") {
		t.Fatal("the two chosen books moved and the third stayed")
	}
	newID := catalog.BookID("Dune/Dune 1.epub")
	if override, ok := server.metadataOverrides.Get("shelf", newID); !ok || override.Title != "Dune (corrected)" {
		t.Fatalf("the correction should follow the book: %v %v", override, ok)
	}
	if _, ok := server.metadataOverrides.Get("books", "a1"); ok {
		t.Fatal("a move leaves nothing behind in the old library")
	}
	if position, ok := server.reading.Get("default", "shelf", newID); !ok || position.Fraction != .4 || position.BookID != newID {
		t.Fatalf("the reading place should follow the book: %v %v", position, ok)
	}
	if _, ok := server.reading.Get("default", "books", "a1"); ok {
		t.Fatal("the old place should be gone")
	}
	// progress made after the move is not lost by undoing it
	if _, err := server.reading.Put("default", reading.Position{LibraryID: "shelf", BookID: catalog.BookID("n.epub"), Fraction: .7}, 0); err != nil {
		t.Fatal(err)
	}
	undo := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+result.Token+`"}`)
	if undo.Code != http.StatusOK {
		t.Fatalf("undo = %d %s", undo.Code, undo.Body.String())
	}
	if !present(one, "n.epub") || !present(one, "Dune/Dune 1.epub") || present(two, "n.epub") || present(two, "Dune") {
		t.Fatal("undo puts the books back and tidies the other library")
	}
	if position, ok := server.reading.Get("default", "books", "a1"); !ok || position.Fraction != .4 {
		t.Fatalf("undo returns the reading place: %v %v", position, ok)
	}
	if position, ok := server.reading.Get("default", "books", "n1"); ok && position.Fraction != 0 && position.Fraction != .7 {
		t.Fatalf("unexpected: %v", position)
	}
}

func TestTransferCopyKeepsTheOriginalAndAClashChangesNothing(t *testing.T) {
	server, one, two := transferServer(t)
	if err := os.WriteFile(filepath.Join(two, "n.epub"), []byte("someone else"), 0o644); err != nil {
		t.Fatal(err)
	}
	clash := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1","n1"],"mode":"copy"}`)
	if clash.Code != http.StatusOK || !strings.Contains(clash.Body.String(), `"blocked":1`) || !strings.Contains(clash.Body.String(), "already has") || present(two, "Dune/Dune 1.epub") {
		t.Fatalf("a clash is reported and nothing is copied: %d %s", clash.Code, clash.Body.String())
	}
	ok := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1"],"mode":"copy"}`)
	if ok.Code != http.StatusOK || !present(one, "Dune/Dune 1.epub") || !present(two, "Dune/Dune 1.epub") {
		t.Fatalf("copy = %d %s", ok.Code, ok.Body.String())
	}
	if _, found := server.reading.Get("default", "shelf", catalog.BookID("Dune/Dune 1.epub")); found {
		t.Fatal("a copy is a new book with no reading place")
	}
}

func TestTransferRefusesLibrariesThatAreNotFolders(t *testing.T) {
	server, _, _ := transferServer(t)
	server.libraries = append(server.libraries, Library{ID: "cal", Name: "Calibre", Kind: "books", SourceType: "calibre", SourcePath: t.TempDir()})
	for _, body := range []string{
		`{"library":"books","to":"cal","ids":["a1"],"mode":"copy"}`,
		`{"library":"books","to":"books","ids":["a1"],"mode":"copy"}`,
		`{"library":"books","to":"nowhere","ids":["a1"],"mode":"copy"}`,
		`{"library":"books","to":"shelf","ids":[],"mode":"copy"}`,
		`{"library":"books","to":"shelf","ids":["a1"],"mode":"shred"}`,
	} {
		if response := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", body); response.Code < 400 {
			t.Fatalf("%s should be refused, got %d", body, response.Code)
		}
	}
}

// A move carries the reader's own writing with the book. Without the step the
// highlights stay under the old library's key: they are invisible in the new
// library, and a later file that gets the same book id in the old one inherits
// them. A copy is a new book, so it starts clean, like its reading place.
func TestAMoveCarriesHighlightsAndNotesWithTheBook(t *testing.T) {
	server, _, _ := transferServer(t)
	store, err := annotations.Open("")
	if err != nil {
		t.Fatal(err)
	}
	server.annotations = store
	if _, err := store.Apply("default", "books", "a1", []annotations.Change{{Annotation: annotations.Annotation{ID: "x1", Kind: "highlight", Quote: "fear is the mind-killer", Note: "litany"}}}); err != nil {
		t.Fatal(err)
	}
	done := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1"],"mode":"move"}`)
	if done.Code != http.StatusOK {
		t.Fatalf("move = %d %s", done.Code, done.Body.String())
	}
	newID := catalog.BookID("Dune/Dune 1.epub")
	if carried := store.List("default", "shelf", newID, 0); len(carried.Applied) != 1 || carried.Applied[0].Quote != "fear is the mind-killer" || carried.Applied[0].Note != "litany" {
		t.Fatalf("the highlight should follow the book: %+v", carried.Applied)
	}
	if left := store.List("default", "books", "a1", 0); len(left.Applied) != 0 {
		t.Fatalf("a move leaves nothing behind under the old book: %+v", left.Applied)
	}
	var moved struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(done.Body.Bytes(), &moved); err != nil || moved.Token == "" {
		t.Fatalf("the move answered no undo token: %s", done.Body.String())
	}
	if undo := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+moved.Token+`"}`); undo.Code != http.StatusOK {
		t.Fatalf("undo = %d %s", undo.Code, undo.Body.String())
	}
	if back := store.List("default", "books", "a1", 0); len(back.Applied) != 1 || back.Applied[0].Quote != "fear is the mind-killer" {
		t.Fatalf("undo should bring the highlight back with the book: %+v", back.Applied)
	}
	if gone := store.List("default", "shelf", newID, 0); len(gone.Applied) != 0 {
		t.Fatalf("undo should leave nothing behind in the other library: %+v", gone.Applied)
	}
}

func TestACopyLeavesTheSourceHighlightsAloneAndStartsTheCopyClean(t *testing.T) {
	server, _, _ := transferServer(t)
	store, err := annotations.Open("")
	if err != nil {
		t.Fatal(err)
	}
	server.annotations = store
	if _, err := store.Apply("default", "books", "a1", []annotations.Change{{Annotation: annotations.Annotation{ID: "x1", Kind: "note", Note: "quiet"}}}); err != nil {
		t.Fatal(err)
	}
	if done := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1"],"mode":"copy"}`); done.Code != http.StatusOK {
		t.Fatalf("copy = %d %s", done.Code, done.Body.String())
	}
	if kept := store.List("default", "books", "a1", 0); len(kept.Applied) != 1 {
		t.Fatalf("the source keeps its note: %+v", kept.Applied)
	}
	if copied := store.List("default", "shelf", catalog.BookID("Dune/Dune 1.epub"), 0); len(copied.Applied) != 0 {
		t.Fatalf("a copy is a new book with no notes: %+v", copied.Applied)
	}
}
