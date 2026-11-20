package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inspectTestHandler(t *testing.T, roots []string, libraries ...Library) http.Handler {
	t.Helper()
	handler, err := NewMulti(MultiConfig{AllowOpen: true, BrowseRoots: roots}, libraries, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func inspectFolder(t *testing.T, handler http.Handler, path string) (*httptest.ResponseRecorder, libraryInspection) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/library-inspect?path="+url.QueryEscape(path), nil))
	var inspection libraryInspection
	if response.Code == http.StatusOK {
		if err := json.Unmarshal(response.Body.Bytes(), &inspection); err != nil {
			t.Fatalf("inspect %q returned invalid JSON: %v (%s)", path, err, response.Body.String())
		}
	}
	return response, inspection
}

func writeTestFiles(t *testing.T, base string, names ...string) {
	t.Helper()
	for _, name := range names {
		full := filepath.Join(base, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLibraryInspectSuggestsFromFolderContents(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		folder   string
		files    []string
		kind     string
		storage  string
		counts   libraryFileCounts
		evidence []string
	}{
		{
			folder: "Calibre Books", kind: "books", storage: "calibre",
			files:    []string{"metadata.db", "Author/Title (1)/Title - Author.epub"},
			counts:   libraryFileCounts{Books: 1},
			evidence: []string{"Calibre database found"},
		},
		{
			folder: "paperbacks", kind: "books", storage: "filesystem",
			files:    []string{"one.epub", "two.epub", "three.PDF", "notes.txt"},
			counts:   libraryFileCounts{Books: 3},
			evidence: []string{"No Calibre database", "3 book files"},
		},
		{
			folder: "comics", kind: "comics", storage: "filesystem",
			files:    []string{"one.cbz", "two.CBR", "cover.png"},
			counts:   libraryFileCounts{Comics: 2, Images: 1},
			evidence: []string{"2 comic files", "1 image"},
		},
		{
			folder: "movies", kind: "movies", storage: "filesystem",
			files:    []string{"one.mp4", "two.mkv", "three.webm"},
			counts:   libraryFileCounts{Video: 3},
			evidence: []string{"3 video files"},
		},
		{
			folder: "audiobooks", kind: "audiobooks", storage: "filesystem",
			files:    []string{"one.mp3", "two.m4b", "three.flac", "four.ogg"},
			counts:   libraryFileCounts{Audio: 4},
			evidence: []string{"4 audio files"},
		},
		{
			folder: "vault", kind: "vault", storage: "filesystem",
			files:    []string{"note.md", "nested/other.markdown"},
			counts:   libraryFileCounts{Documents: 2},
			evidence: []string{"2 markdown notes"},
		},
		{
			folder: "emptyshelf", kind: "books", storage: "filesystem",
			counts:   libraryFileCounts{},
			evidence: []string{"Empty folder"},
		},
		{
			folder: "unreadable", kind: "books", storage: "filesystem",
			files:    []string{"readme.txt"},
			counts:   libraryFileCounts{},
			evidence: []string{"No library files"},
		},
	}
	for _, test := range tests {
		t.Run(test.folder, func(t *testing.T) {
			folder := filepath.Join(root, test.folder)
			if err := os.MkdirAll(folder, 0o755); err != nil {
				t.Fatal(err)
			}
			writeTestFiles(t, folder, test.files...)
			handler := inspectTestHandler(t, []string{root})
			response, inspection := inspectFolder(t, handler, folder)
			if response.Code != http.StatusOK {
				t.Fatalf("inspect = %d %s", response.Code, response.Body.String())
			}
			if inspection.Path != folder {
				t.Fatalf("path = %q, want %q", inspection.Path, folder)
			}
			want := librarySuggestion{
				Name:    test.folder,
				Kind:    test.kind,
				Storage: test.storage,
			}
			if inspection.Suggested != want {
				t.Fatalf("suggested = %+v, want %+v", inspection.Suggested, want)
			}
			if len(inspection.Evidence) == 0 {
				t.Fatal("evidence must explain the suggestion")
			}
			joined := strings.Join(inspection.Evidence, "\n")
			for _, fragment := range test.evidence {
				if !strings.Contains(joined, fragment) {
					t.Fatalf("evidence %q does not mention %q", joined, fragment)
				}
			}
		})
	}
}

func TestLibraryInspectRefusesPathsItMayNotRead(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	writeTestFiles(t, filepath.Join(root, "books"), "one.epub")
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	handler := inspectTestHandler(t, []string{root})
	for _, path := range []string{"", "relative/path", "/etc", outside, filepath.Join(root, "escape"), filepath.Join(root, "missing"), filepath.Join(root, "books", "one.epub")} {
		response, _ := inspectFolder(t, handler, path)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("inspect %q = %d, want 400 (%s)", path, response.Code, response.Body.String())
		}
		if strings.TrimSpace(response.Body.String()) == "" {
			t.Fatalf("inspect %q must explain the refusal", path)
		}
	}
}

func TestLibraryInspectIgnoresHiddenFoldersAndSymlinkEscapes(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	library := filepath.Join(root, "library")
	writeTestFiles(t, library, "one.epub", ".hidden/two.cbz", "nested/three.epub")
	writeTestFiles(t, outside, "movie.mkv")
	if err := os.Symlink(outside, filepath.Join(library, "linked")); err != nil {
		t.Fatal(err)
	}
	handler := inspectTestHandler(t, []string{root})
	response, inspection := inspectFolder(t, handler, library)
	if response.Code != http.StatusOK {
		t.Fatalf("inspect = %d %s", response.Code, response.Body.String())
	}
	if inspection.Suggested.Kind != "books" {
		t.Fatalf("kind = %q", inspection.Suggested.Kind)
	}
	joined := strings.Join(inspection.Evidence, "\n")
	if !strings.Contains(joined, "2 book files") || strings.Contains(joined, "comic") {
		t.Fatalf("evidence = %q, want only the visible book (a hidden folder and a symlink out of the folder are not the library)", joined)
	}
}

func TestLibraryInspectSuggestsTheFolderNameWithoutClaimingAnID(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "books")
	writeTestFiles(t, folder, "one.epub")
	handler := inspectTestHandler(t, []string{root},
		Library{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}},
		Library{ID: "books-2", Name: "More books", Kind: "books", Store: staticBackend{}},
	)
	response, inspection := inspectFolder(t, handler, folder)
	if response.Code != http.StatusOK {
		t.Fatalf("inspect = %d %s", response.Code, response.Body.String())
	}
	if inspection.Suggested.Name != "books" {
		t.Fatalf("name = %q, the folder name stays readable", inspection.Suggested.Name)
	}
	// The settings page derives the id from the name the user settles on, so a
	// suggestion with its own id would be a second, divergent rule. Counts are
	// only what the evidence line is built from; nothing reads them.
	for _, dead := range []string{`"id"`, `"counts"`} {
		if strings.Contains(response.Body.String(), dead) {
			t.Fatalf("payload carries %s, which no client reads: %s", dead, response.Body.String())
		}
	}
}

func TestLibraryInspectNamesTheLibraryRoot(t *testing.T) {
	for _, path := range []string{"", "/", ".", string(filepath.Separator)} {
		if got := suggestedLibraryName(path); got != "Library" {
			t.Fatalf("suggestedLibraryName(%q) = %q, want Library", path, got)
		}
	}
	if got := suggestedLibraryName("/media/books/"); got != "books" {
		t.Fatalf("suggestedLibraryName = %q", got)
	}
}

func TestLibraryInspectRequiresAnAdministrator(t *testing.T) {
	if !administratorOnly("/api/library-inspect", http.MethodGet) {
		t.Fatal("reading a folder's contents is an administrator operation, like /api/folders")
	}
	if administratorOnly("/api/library-inspect", http.MethodPost) != true {
		t.Fatal("every method of the inspector is administrative")
	}
	handler := inspectTestHandler(t, []string{t.TempDir()})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/library-inspect", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d, want 405", response.Code)
	}
}

func TestLibraryInspectBoundsHowDeepItWalks(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "library")
	writeTestFiles(t, filepath.Join(library, strings.Repeat("d/", maximumInspectDepth-1)), "found.epub")
	writeTestFiles(t, filepath.Join(library, strings.Repeat("d/", maximumInspectDepth+2)), "unreachable.epub")
	handler := inspectTestHandler(t, []string{root})
	response, inspection := inspectFolder(t, handler, library)
	if response.Code != http.StatusOK {
		t.Fatalf("inspect = %d %s", response.Code, response.Body.String())
	}
	joined := strings.Join(inspection.Evidence, "\n")
	if !strings.Contains(joined, "1 book file") || strings.Contains(joined, "2 book files") {
		t.Fatalf("evidence = %q, want only the file within the depth cap", joined)
	}
}

// The settings card reads response.data.suggested.*, so the wire names matter.
func TestLibraryInspectResponseShape(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "books")
	writeTestFiles(t, folder, "one.epub")
	handler := inspectTestHandler(t, []string{root})
	response, _ := inspectFolder(t, handler, folder)
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("inspect body: %v", err)
	}
	for _, key := range []string{"path", "suggested", "evidence"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("inspect body is missing %q: %s", key, response.Body.String())
		}
	}
	suggested, _ := body["suggested"].(map[string]any)
	for _, key := range []string{"name", "kind", "storage"} {
		if _, ok := suggested[key]; !ok {
			t.Fatalf("suggested is missing %q: %v", key, suggested)
		}
	}
	// The id and the raw counts were never read by the settings page: the card
	// derives the id from the name, and the counts only feed the evidence line.
	for _, dead := range []string{"counts"} {
		if _, ok := body[dead]; ok {
			t.Fatalf("inspect body still carries %q: %s", dead, response.Body.String())
		}
	}
	if _, ok := suggested["id"]; ok {
		t.Fatalf("suggested still carries an id: %v", suggested)
	}
}

func TestLibraryInspectDoesNotMentionCalibreInAFolderOfVideos(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "movies")
	writeTestFiles(t, folder, "one.mkv", "two.mkv", "three.mkv")
	handler := inspectTestHandler(t, []string{root})
	response, inspection := inspectFolder(t, handler, folder)
	if response.Code != http.StatusOK {
		t.Fatalf("inspect = %d %s", response.Code, response.Body.String())
	}
	joined := strings.Join(inspection.Evidence, "\n")
	if strings.Contains(joined, "Calibre") {
		t.Fatalf("evidence = %q, a video folder was never a Calibre library", joined)
	}
	if !strings.Contains(joined, "3 video files") {
		t.Fatalf("evidence = %q, want the video count", joined)
	}
}
