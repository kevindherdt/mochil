package web

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestTheEditorSaysWhereTheLibraryLivesAndCanSortByPath(t *testing.T) {
	server := editorTestServer(t)
	server.libraries[0].SourcePath = "/media/calibre/books"
	response := editorCall(server, http.MethodGet, "/api/metadata/editor/books?library=books&sort=path&dir=desc", "")
	var body struct {
		LibraryFolder string      `json:"library_folder"`
		Items         []editorRow `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusOK {
		t.Fatalf("%d %v", response.Code, err)
	}
	if body.LibraryFolder != "/media/calibre/books" {
		t.Fatalf("library folder = %q", body.LibraryFolder)
	}
	paths := make([]string, len(body.Items))
	for i, item := range body.Items {
		paths[i] = item.Path
	}
	for i := 1; i < len(paths); i++ {
		if paths[i-1] < paths[i] {
			t.Fatalf("descending by path, got %v", paths)
		}
	}
}

func TestEveryEditorColumnCanBeSorted(t *testing.T) {
	rows := []editorRow{
		{ID: "a", Title: "A", Identifiers: "isbn:2", TitleSort: "b", AuthorSort: "y", Description: "zeta", IsbnFound: "222", Match: &matchCandidate{Score: 0.5}, Issues: []string{"x", "y"}},
		{ID: "b", Title: "B", Identifiers: "isbn:1", TitleSort: "a", AuthorSort: "x", Description: "alpha", IsbnFound: "111", Match: &matchCandidate{Score: 0.9}, Issues: []string{"x"}},
		{ID: "c", Title: "C"},
	}
	want := map[string]string{"identifiers": "bac", "title_sort": "bac", "author_sort": "bac", "description": "bac", "isbn_found": "bac", "match": "abc", "status": "cba"}
	for column, order := range want {
		sorted := append([]editorRow(nil), rows...)
		sortEditorRows(sorted, column, false)
		got := ""
		for _, row := range sorted {
			got += row.ID
		}
		if got != order {
			t.Errorf("sorting by %s gives %s, want %s", column, got, order)
		}
	}
}
