package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mochil/internal/audio"
)

// Settings -> Libraries lets you drag libraries into the order you want them in the switcher, and
// that order is nothing but the order of the array in the settings, which becomes the order in
// config.json, which becomes the order handed to the query service. Every one of those steps has to
// preserve it, and nothing in the chain sorts for itself. This pins the last step: /api/libraries
// returns what it was given, in the order it was given.
//
// If someone later "tidies" the list by sorting it, the drag ordering would silently stop doing
// anything while the settings screen still claimed to have saved it.
func TestLibrariesAreListedInTheConfiguredOrder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Deliberately not alphabetical, and not the order a scan would produce.
	libraries := []Library{
		{ID: "zebra", Name: "Zebra", Kind: "books", Store: staticBackend{}},
		{ID: "alpha", Name: "Alpha", Kind: "books", Store: staticBackend{}},
		{ID: "middle", Name: "Middle", Kind: "books", Store: staticBackend{}},
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, libraries, logger)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/libraries", nil))
	if response.Code != 200 {
		t.Fatalf("libraries: %d %s", response.Code, response.Body.String())
	}
	var listed []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode: %v (%s)", err, response.Body.String())
	}
	var order []string
	for _, library := range listed {
		order = append(order, library.ID)
	}
	want := []string{"zebra", "alpha", "middle"}
	if len(order) != len(want) {
		t.Fatalf("listed %d libraries, want %d: %v", len(order), len(want), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("libraries were reordered: got %v, want %v", order, want)
		}
	}
}

// A music, video or notes library used to be forced below every book library however the order was
// set, because they were appended after the publications rather than listed with them. The drag
// worked, the save worked, and the shelf still sat at the bottom.
func TestAnAudioLibraryKeepsItsPlaceAmongTheBookLibraries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Track.mp3"), []byte("synthetic audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}

	libraries := []Library{
		{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}},
		{ID: "music", Name: "Music", Kind: "music", Audio: store},
		{ID: "notes", Name: "Notes", Kind: "documents", Store: staticBackend{}},
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true}, libraries, logger)
	if err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/libraries", nil))
	if response.Code != 200 {
		t.Fatalf("libraries: %d %s", response.Code, response.Body.String())
	}
	var listed []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []string{"books", "music", "notes"}
	if len(listed) != len(want) {
		t.Fatalf("listed %d libraries, want %d", len(listed), len(want))
	}
	for i, id := range want {
		if listed[i].ID != id {
			t.Fatalf("order is %v, want %v", listed, want)
		}
	}
}
