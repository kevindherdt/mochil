package web

import (
	"encoding/json"
	"net/http"
	"testing"
)

func editorValuesFor(t *testing.T, path string) (int, []editorSuggestion) {
	t.Helper()
	response := editorCall(editorTestServer(t), http.MethodGet, path, "")
	var values []editorSuggestion
	_ = json.Unmarshal(response.Body.Bytes(), &values)
	return response.Code, values
}

func TestEditorSuggestsNamesThatBeginWithWhatIsTypedBeforeOnesThatContainIt(t *testing.T) {
	code, values := editorValuesFor(t, "/api/metadata/editor/values?library=books&field=authors&q=an")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	// "Unknown" contains "an"? no: only names with the letters appear. Frank Herbert contains "an".
	if len(values) != 1 || values[0].Name != "Frank Herbert" || values[0].Count != 2 {
		t.Fatalf("values = %+v, want Frank Herbert counted twice", values)
	}
	_, values = editorValuesFor(t, "/api/metadata/editor/values?library=books&field=series&q=d")
	if len(values) != 1 || values[0].Name != "Dune" || values[0].Count != 2 {
		t.Fatalf("series = %+v", values)
	}
}

func TestEditorSuggestionsRankBeginningsFirst(t *testing.T) {
	counts := map[string]int{"Brandon Sanderson": 11, "Andy Weir": 1, "Alan B. Lloyd": 1, "Andrew Rowe": 9}
	got := rankedValues(counts, "an", 8)
	want := []string{"Andrew Rowe", "Andy Weir", "Alan B. Lloyd", "Brandon Sanderson"}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("got %+v, want order %v", got, want)
		}
	}
}

func TestEditorSuggestionsRefuseAFieldTheyDoNotKnow(t *testing.T) {
	if code, _ := editorValuesFor(t, "/api/metadata/editor/values?library=books&field=description&q=x"); code != http.StatusBadRequest {
		t.Fatalf("status %d", code)
	}
}
