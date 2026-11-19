package web

import (
	"net/http"
	"testing"
)

func TestEditorRefusesMetadataCalibreOwns(t *testing.T) {
	server := editorTestServer(t)
	server.libraries[0].SourceType = "calibre"

	refused := editorCall(server, http.MethodPost, "/api/metadata/editor/edit",
		`{"library":"books","changes":{"a1":{"title":"A title Calibre would overwrite"}}}`)
	if refused.Code != http.StatusConflict {
		t.Fatalf("calibre title edit = %d, want %d: %s", refused.Code, http.StatusConflict, refused.Body.String())
	}

	allowed := editorCall(server, http.MethodPost, "/api/metadata/editor/edit",
		`{"library":"books","changes":{"a1":{"rating":"4"}}}`)
	if allowed.Code != http.StatusOK {
		t.Fatalf("calibre rating edit = %d, want 200: %s", allowed.Code, allowed.Body.String())
	}

	reset := editorCall(server, http.MethodPost, "/api/metadata/editor/edit",
		`{"library":"books","reset":["a1"]}`)
	if reset.Code != http.StatusConflict {
		t.Fatalf("calibre reset = %d, want %d: %s", reset.Code, http.StatusConflict, reset.Body.String())
	}
}

func TestEditorStillEditsMetadataInANonCalibreLibrary(t *testing.T) {
	server := editorTestServer(t)
	edit := editorCall(server, http.MethodPost, "/api/metadata/editor/edit",
		`{"library":"books","changes":{"a1":{"title":"A title Mochil owns"}}}`)
	if edit.Code != http.StatusOK {
		t.Fatalf("plain library edit = %d, want 200: %s", edit.Code, edit.Body.String())
	}
}

// The guard must cover the fields Calibre's own database holds, not the narrower list a Hardcover lookup is built from:
// the series was refused while the volume number beside it (and the sort keys) walked through and shadowed Calibre.
func TestEditorRefusesEveryCalibreOwnedFieldNotJustTheLookupInputs(t *testing.T) {
	server := editorTestServer(t)
	server.libraries[0].SourceType = "calibre"
	for _, field := range []struct{ name, body string }{
		{"series_index", `{"library":"books","changes":{"a1":{"series_index":"9"}}}`},
		{"title_sort", `{"library":"books","changes":{"a1":{"title_sort":"Herbert, Frank"}}}`},
		{"author_sort", `{"library":"books","changes":{"a1":{"author_sort":"Herbert, Frank"}}}`},
	} {
		response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", field.body)
		if response.Code != http.StatusConflict {
			t.Fatalf("calibre %s edit = %d, want %d: %s", field.name, response.Code, http.StatusConflict, response.Body.String())
		}
	}
	// The fields the editor is meant to keep editable on a Calibre library stay editable.
	for _, field := range []struct{ name, body string }{
		{"rating", `{"library":"books","changes":{"a1":{"rating":"4"}}}`},
		{"tags", `{"library":"books","changes":{"a1":{"tags":"space opera"}}}`},
		{"publisher", `{"library":"books","changes":{"a1":{"publisher":"Ace"}}}`},
	} {
		response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", field.body)
		if response.Code != http.StatusOK {
			t.Fatalf("calibre %s edit = %d, want 200: %s", field.name, response.Code, response.Body.String())
		}
	}
}
