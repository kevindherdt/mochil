package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// matchFieldsServer is the editor's test library with a match-fields store that has a real file behind it,
// so a read after a write proves what a restart would see rather than what memory happens to hold.
func matchFieldsServer(t *testing.T) *multiServer {
	t.Helper()
	server := editorTestServer(t)
	store, err := openMatchFields(filepath.Join(t.TempDir(), "match-fields.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.matchFields = store
	return server
}

func readMatchFields(t *testing.T, server *multiServer, library string) []string {
	t.Helper()
	response := editorCall(server, http.MethodGet, "/api/metadata/editor/match-fields?library="+library, "")
	if response.Code != http.StatusOK {
		t.Fatalf("match fields = %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Fields []string `json:"fields"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("match fields are not readable JSON: %v %s", err, response.Body.String())
	}
	return payload.Fields
}

func saveMatchFields(t *testing.T, server *multiServer, body string) {
	t.Helper()
	response := editorCall(server, http.MethodPut, "/api/metadata/editor/match-fields", body)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("save = %d %q, want 204 with no body", response.Code, response.Body.String())
	}
}

// A library that has never chosen anything gets every field a match can carry, which is what taking a match
// meant before this setting existed. The order is the editor's own, so the boxes have one order everywhere.
func TestMatchFieldsStartWithEveryFieldInCanonicalOrder(t *testing.T) {
	server := matchFieldsServer(t)
	want := []string{"title", "authors", "series", "series_index", "published", "description", "identifiers"}
	got := readMatchFields(t, server, "books")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default = %v, want %v", got, want)
	}
	// The payload names the key the client reads, and an empty library still carries the list rather than
	// leaving the client to invent a default.
	raw := editorCall(server, http.MethodGet, "/api/metadata/editor/match-fields?library=books", "").Body.String()
	if !strings.Contains(raw, `"fields":[`) {
		t.Fatalf("the payload does not carry the field list: %s", raw)
	}
}

// Saving replaces the whole list and reads back through the file, so a restart keeps the choice, and an
// empty list stays an empty list: it means "a match writes nothing", not "never chosen".
func TestMatchFieldsSaveThenRead(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want []string
	}{
		{
			name: "a subset is kept in the canonical order, whatever order it was sent in",
			body: `{"library":"books","fields":["description","title"]}`,
			want: []string{"title", "description"},
		},
		{
			name: "a repeated field is kept once",
			body: `{"library":"books","fields":["series","title","series"]}`,
			want: []string{"title", "series"},
		},
		{
			name: "an empty list means a match writes nothing",
			body: `{"library":"books","fields":[]}`,
			want: []string{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := matchFieldsServer(t)
			saveMatchFields(t, server, test.body)
			if got := readMatchFields(t, server, "books"); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("after saving %s the answer is %v, want %v", test.body, got, test.want)
			}
			// A restart is the point of the file: open it again and ask the same question.
			reopened, err := openMatchFields(server.matchFields.path)
			if err != nil {
				t.Fatal(err)
			}
			if got := reopened.fields("books"); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("the file holds %v, want %v", got, test.want)
			}
			if len(test.want) == 0 {
				raw := editorCall(server, http.MethodGet, "/api/metadata/editor/match-fields?library=books", "").Body.String()
				if !strings.Contains(raw, `"fields":[]`) {
					t.Fatalf("an empty choice reads as something else: %s", raw)
				}
			}
		})
	}
}

// Library ids are not unique in what they mean: books and comics are edited separately, and pinning a series
// name in one must not pin it in the other.
func TestMatchFieldsAreKeptPerLibrary(t *testing.T) {
	server := matchFieldsServer(t)
	server.libraries = append(server.libraries, Library{
		ID: "comics", Name: "Comics", Kind: "books",
		Store: server.metadataOverrides.Wrap("comics", staticBackend{books: server.libraries[0].Store.Books()}),
	})
	saveMatchFields(t, server, `{"library":"books","fields":["description"]}`)
	if got := readMatchFields(t, server, "books"); !reflect.DeepEqual(got, []string{"description"}) {
		t.Fatalf("books = %v, want [description]", got)
	}
	want := []string{"title", "authors", "series", "series_index", "published", "description", "identifiers"}
	if got := readMatchFields(t, server, "comics"); !reflect.DeepEqual(got, want) {
		t.Fatalf("comics = %v, want every field", got)
	}
	reopened, err := openMatchFields(server.matchFields.path)
	if err != nil {
		t.Fatal(err)
	}
	if _, stored := reopened.libraries["comics"]; stored {
		t.Fatal("saving for books wrote a row for comics too")
	}
}

// A field name the editor cannot write is a refusal, not a silently dropped member: a typo that saved would
// look like the choice was honoured while one field quietly did nothing. Nothing is written on the way out.
func TestAnUnknownMatchFieldIsRefusedAndStoresNothing(t *testing.T) {
	server := matchFieldsServer(t)
	response := editorCall(server, http.MethodPut, "/api/metadata/editor/match-fields", `{"library":"books","fields":["title","publisher"]}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown field = %d %s, want 400", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "publisher") {
		t.Fatalf("the refusal does not say which field: %s", response.Body.String())
	}
	want := []string{"title", "authors", "series", "series_index", "published", "description", "identifiers"}
	if got := readMatchFields(t, server, "books"); !reflect.DeepEqual(got, want) {
		t.Fatalf("a refused request changed the answer to %v", got)
	}
	if _, err := os.Stat(server.matchFields.path); !os.IsNotExist(err) {
		t.Fatalf("a refused request wrote a file: %v", err)
	}
}

// A body without a fields member is not the same as an empty list: one means the client forgot, the other
// means the person unticked everything, and guessing between them would either wipe the choice or widen it.
func TestAMatchFieldsRequestMustNameItsFields(t *testing.T) {
	server := matchFieldsServer(t)
	response := editorCall(server, http.MethodPut, "/api/metadata/editor/match-fields", `{"library":"books"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing fields = %d %s, want 400", response.Code, response.Body.String())
	}
	if _, err := os.Stat(server.matchFields.path); !os.IsNotExist(err) {
		t.Fatalf("a refused request wrote a file: %v", err)
	}
}

// The library is validated the same way every other editor endpoint validates it: a name that is not a
// publication library is a 404 before anything is read or written.
func TestMatchFieldsRefuseALibraryThatCannotBeEdited(t *testing.T) {
	server := matchFieldsServer(t)
	for _, path := range []string{
		"/api/metadata/editor/match-fields?library=nowhere",
	} {
		if response := editorCall(server, http.MethodGet, path, ""); response.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d %s, want 404", path, response.Code, response.Body.String())
		}
	}
	if response := editorCall(server, http.MethodPut, "/api/metadata/editor/match-fields", `{"library":"nowhere","fields":[]}`); response.Code != http.StatusNotFound {
		t.Fatalf("PUT for an unknown library = %d %s, want 404", response.Code, response.Body.String())
	}
}

// The file is replaced, never written in place, and nothing is left beside it: a crash mid-write must leave
// the previous choice readable rather than half a list, and a stray temp file would be a second source of
// truth for the next person to find.
func TestTheMatchFieldsFileIsReplacedAtomically(t *testing.T) {
	server := matchFieldsServer(t)
	saveMatchFields(t, server, `{"library":"books","fields":["title"]}`)
	saveMatchFields(t, server, `{"library":"books","fields":["description","published"]}`)
	entries, err := os.ReadDir(filepath.Dir(server.matchFields.path))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !reflect.DeepEqual(names, []string{"match-fields.json"}) {
		t.Fatalf("the directory holds %v, want only match-fields.json", names)
	}
	info, err := os.Stat(server.matchFields.path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("the file is %o, want 600", mode)
	}
	var stored matchFieldFile
	data, err := os.ReadFile(server.matchFields.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("the file is not readable JSON: %v %s", err, data)
	}
	if stored.Version != matchFieldsVersion {
		t.Fatalf("the file says version %d, want %d", stored.Version, matchFieldsVersion)
	}
	if !strings.Contains(string(data), `"libraries"`) {
		t.Fatalf("the file has no libraries map: %s", data)
	}
}
