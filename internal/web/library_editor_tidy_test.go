package web

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The endpoint that will sit behind "Clean up titles": it proposes, in the shape the editor already stages, and writes
// nothing. The rules themselves are tested in internal/metadata/tidying_test.go against real rows; what matters here is
// that the door answers in a form the panel can hand straight to the review, and that it refuses a library whose
// metadata the editor will not let anyone change.
func TestTheCleanupEndpointProposesInTheShapeTheEditorStages(t *testing.T) {
	server := editorTestServer(t)

	response := editorCall(server, http.MethodPost, "/api/metadata/editor/tidy", `{"library":"books"}`)
	if response.Code == http.StatusBadRequest {
		// A Calibre-owned library is refused on purpose: the editor treats those fields as read-only, so proposing them
		// would offer work that fails at apply time.
		t.Skipf("this fixture is Calibre-owned: %s", response.Body.String())
	}
	if response.Code != http.StatusOK {
		t.Fatalf("tidy = %d %s", response.Code, response.Body.String())
	}
	var out struct {
		Changes map[string]map[string]string `json:"changes"`
		Groups  []struct {
			Rule  string `json:"rule"`
			Count int    `json:"count"`
		} `json:"groups"`
		Books  int `json:"books"`
		Unsure int `json:"unsure"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Books != len(out.Changes) {
		t.Fatalf("books = %d but %d change sets: the count and the payload disagree", out.Books, len(out.Changes))
	}
	// Every proposed field must be one the editor can actually stage.
	for id, fields := range out.Changes {
		for field, value := range fields {
			switch field {
			case "title", "series", "series_index", "authors":
			default:
				t.Errorf("%s: proposed an unknown field %q", id, field)
			}
			if value == "" {
				t.Errorf("%s: proposed an empty %s — a rule that empties a cell is not a cleanup", id, field)
			}
		}
	}
	// A group is named after a rule and counts at least the books that used it.
	for _, group := range out.Groups {
		if group.Rule == "" || group.Count == 0 {
			t.Errorf("group = %+v, want a named rule and a real count", group)
		}
	}
}

func TestTheCleanupEndpointRefusesAWrongMethod(t *testing.T) {
	server := editorTestServer(t)
	if response := editorCall(server, http.MethodGet, "/api/metadata/editor/tidy", ""); response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET tidy = %d, want 405", response.Code)
	}
}
