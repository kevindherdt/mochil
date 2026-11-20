package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/narration"
)

func narrationBookServer(t *testing.T, modules map[string]bool) (*multiServer, catalog.Book) {
	t.Helper()
	root := t.TempDir()
	writeTestEPUB(t, filepath.Join(root, "One.epub"),
		`<h1>Chapter One</h1><p>The lantern swung over the harbour. The old captain counted every rope twice. Nobody spoke.</p><p>Copyright © Someone 2014</p>`,
		`<p>A second chapter with its own words to read aloud.</p>`)
	store := catalog.NewFilesystemStore(root, filepath.Join(t.TempDir(), "cache.json"), catalog.FilesystemOptions{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		Modules: modules, Narration: narration.Config{Python: fakeNarrationRuntime(t)},
		NarrationVoices: filepath.Join(t.TempDir(), "voices"), ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	m := handler.(*managedHandler).Handler.(*multiServer)
	t.Cleanup(func() {
		if m.narration != nil {
			m.narration.Close()
		}
	})
	return m, store.Books()[0]
}

func get(m *multiServer, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestABookIsReadAloudChapterBySentenceWithAheadWorkDone(t *testing.T) {
	m, book := narrationBookServer(t, map[string]bool{"narration": true})
	base := "/api/narration/book/books/" + book.ID

	chapters := get(m, base)
	var list struct {
		Title    string `json:"title"`
		Chapters []struct {
			Index int    `json:"index"`
			Href  string `json:"href"`
		} `json:"chapters"`
	}
	if chapters.Code != http.StatusOK || json.Unmarshal(chapters.Body.Bytes(), &list) != nil || len(list.Chapters) != 2 || list.Chapters[1].Href != "c1.xhtml" {
		t.Fatalf("chapters: %d %s", chapters.Code, chapters.Body.String())
	}

	text := get(m, base+"/chapters/0")
	var chapter struct {
		Sentences []struct {
			Text        string  `json:"text"`
			Progression float64 `json:"progression"`
		} `json:"sentences"`
	}
	if text.Code != http.StatusOK || json.Unmarshal(text.Body.Bytes(), &chapter) != nil {
		t.Fatalf("chapter: %d %s", text.Code, text.Body.String())
	}
	var got []string
	for _, s := range chapter.Sentences {
		got = append(got, s.Text)
	}
	// the copyright line is left out
	if strings.Join(got, "|") != "Chapter One|The lantern swung over the harbour.|The old captain counted every rope twice.|Nobody spoke." {
		t.Fatalf("sentences: %q", got)
	}

	audio := get(m, base+"/chapters/0/audio/0?voice=cosette")
	if audio.Code != http.StatusOK || audio.Header().Get("Content-Type") != "audio/wav" || !strings.HasPrefix(audio.Body.String(), "RIFF") {
		t.Fatalf("audio: %d %q", audio.Code, audio.Header().Get("Content-Type"))
	}
	// The sentence after it is made without being asked for.
	revision := strconv.FormatInt(book.Size, 10) + "\x00" + strconv.FormatInt(book.Modified.UnixNano(), 10)
	key := func(i string) string {
		return base2(book.ID) + "\x00" + revision + "\x00" + "0" + "\x00" + i + "\x00" + "cosette"
	}
	deadline := time.Now().Add(5 * time.Second)
	for !narrationClipCache.has(key("1")) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !narrationClipCache.has(key("1")) {
		t.Fatal("the next sentence should have been prepared while the first one played")
	}
}

func base2(bookID string) string { return "books/" + bookID }

func TestNarrationBookRejectsWhatIsNotThere(t *testing.T) {
	m, book := narrationBookServer(t, map[string]bool{"narration": true})
	base := "/api/narration/book/books/" + book.ID
	for path, want := range map[string]int{
		"/api/narration/book/books/nope":                    http.StatusNotFound,
		"/api/narration/book/nolibrary/" + book.ID:          http.StatusNotFound,
		base + "/chapters/9":                                http.StatusNotFound,
		base + "/chapters/x":                                http.StatusNotFound,
		base + "/chapters/0/audio/99?voice=cosette":         http.StatusNotFound,
		base + "/chapters/0/audio/0?voice=../../etc/passwd": http.StatusBadRequest,
		base + "/chapters/0/audio/-1?voice=cosette":         http.StatusNotFound,
	} {
		if got := get(m, path).Code; got != want {
			t.Errorf("%s: got %d, want %d", path, got, want)
		}
	}
	post := httptest.NewRecorder()
	m.ServeHTTP(post, httptest.NewRequest(http.MethodPost, base, nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", post.Code)
	}
}

func TestNarrationBookIsGoneWhenTheModuleIsOff(t *testing.T) {
	m, book := narrationBookServer(t, map[string]bool{"narration": false})
	if got := get(m, "/api/narration/book/books/"+book.ID).Code; got != http.StatusNotFound {
		t.Fatalf("module off: %d", got)
	}
}

func TestTheAndroidAppReadsABookAloudWithItsDeviceToken(t *testing.T) {
	m, book := narrationBookServer(t, map[string]bool{"narration": true})
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.androidDevices = devices
	token, _, err := devices.issue(m.readingAccount(nil), "Fixture phone")
	if err != nil {
		t.Fatal(err)
	}
	ask := func(path, credential string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if credential != "" {
			r.Header.Set("Authorization", "Bearer "+credential)
		}
		w := httptest.NewRecorder()
		m.androidAPI(w, r)
		return w
	}
	base := "/api/android/v1/narration/book/books/" + book.ID
	if got := ask(base, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("without a token: %d", got)
	}
	if got := ask(base, token); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"chapters"`) {
		t.Fatalf("chapters: %d %s", got.Code, got.Body.String())
	}
	if got := ask(base+"/chapters/0", token); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "lantern") {
		t.Fatalf("text: %d %s", got.Code, got.Body.String())
	}
	if got := ask(base+"/chapters/0/audio/1?voice=cosette", token); got.Code != http.StatusOK || !strings.HasPrefix(got.Body.String(), "RIFF") {
		t.Fatalf("audio: %d", got.Code)
	}
	if got := ask("/api/android/v1/narration/status", token); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"cosette"`) {
		t.Fatalf("status: %d %s", got.Code, got.Body.String())
	}
}
