package web

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
)

func writeTestEPUB(t *testing.T, path string, chapters ...string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	add := func(name, content string) {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(content))
	}
	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", `<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`)
	var manifest, spine strings.Builder
	for index, chapter := range chapters {
		manifest.WriteString(fmt.Sprintf(`<item id="c%d" href="c%d.xhtml" media-type="application/xhtml+xml"/>`, index, index))
		spine.WriteString(fmt.Sprintf(`<itemref idref="c%d"/>`, index))
		add(fmt.Sprintf("c%d.xhtml", index), `<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>t</title></head><body>`+chapter+`</body></html>`)
	}
	add("content.opf", `<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>One</dc:title></metadata><manifest>`+manifest.String()+`</manifest><spine>`+spine.String()+`</spine></package>`)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}

// A stock KOReader (no Mochil plugin) reports crengine XPointers. The server
// must turn them into text anchors for the web reader, and hand a newer web
// position back as an XPointer.
func TestStockKoSyncPositionsRoundTripThroughWebReader(t *testing.T) {
	root := t.TempDir()
	writeTestEPUB(t, filepath.Join(root, "One.epub"),
		`<p>A different first chapter with its own wording, long enough to never be confused.</p>`,
		`<div><p>Opening paragraph that is long enough to be unique in this chapter alone.</p><p>The lantern swung over the harbour while the old captain counted every rope twice.</p><p>Closing paragraph after the interesting one, again with plenty of words to read.</p></div>`)
	store := catalog.NewFilesystemStore(root, filepath.Join(t.TempDir(), "cache.json"), catalog.FilesystemOptions{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		KoSyncEnabled: true, KoSyncDatabase: filepath.Join(t.TempDir(), "kosync.json"), KoSyncUsername: "reader", KoSyncPassword: "secret",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	document := koSyncFilenameMD5("One.epub")
	sum := md5.Sum([]byte("secret"))
	key := hex.EncodeToString(sum[:])

	body := fmt.Sprintf(`{"document":%q,"progress":"/body/DocFragment[2]/body/div/p[2]/text().0","percentage":0.6,"device":"Kobo","device_id":"kobo-1"}`, document)
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, koSyncRequest(http.MethodPut, "/syncs/progress", "reader", key, body))
	if put.Code != http.StatusOK {
		t.Fatalf("progress put = %d %s", put.Code, put.Body.String())
	}
	var position struct {
		WebAnchor *struct {
			Exact      string `json:"exact"`
			SpineIndex int    `json:"spine_index"`
		} `json:"web_anchor"`
		Revision int64 `json:"revision"`
	}
	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/reading/books/"+book.ID, nil))
	if read.Code != http.StatusOK || json.Unmarshal(read.Body.Bytes(), &position) != nil || position.WebAnchor == nil ||
		!strings.HasPrefix(position.WebAnchor.Exact, "The lantern swung") || position.WebAnchor.SpineIndex != 1 {
		t.Fatalf("KOReader position did not become a web anchor: %d %s", read.Code, read.Body.String())
	}

	time.Sleep(1100 * time.Millisecond) // KoSync timestamps have one-second precision
	web := fmt.Sprintf(`{"fraction":0.9,"base_revision":%d,"device_id":"web","web_anchor":{"version":2,"prefix":"","exact":"Closing paragraph after the interesting one, again with plenty of words","suffix":"to read.","spine_index":1,"progress":0.9}}`, position.Revision)
	save := httptest.NewRecorder()
	handler.ServeHTTP(save, httptest.NewRequest(http.MethodPut, "/api/reading/books/"+book.ID, strings.NewReader(web)))
	if save.Code != http.StatusOK {
		t.Fatalf("web save = %d %s", save.Code, save.Body.String())
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, koSyncRequest(http.MethodGet, "/syncs/progress/"+document, "reader", key, ""))
	var progress koSyncProgress
	if get.Code != http.StatusOK || json.Unmarshal(get.Body.Bytes(), &progress) != nil || progress.Progress != "/body/DocFragment[2]/body/div/p[3]/text().0" || progress.Percentage != 0.9 {
		t.Fatalf("web position was not offered to KOReader: %d %s", get.Code, get.Body.String())
	}
}

// Android saves a CREngine-style location with an empty text quote. The server
// fills the quote in, so the web reader and KOReader can reopen the same page.
func TestPositionWithoutAQuoteGetsOneFromItsKoReaderLocation(t *testing.T) {
	root := t.TempDir()
	writeTestEPUB(t, filepath.Join(root, "One.epub"),
		`<p>A different first chapter with its own wording, long enough to never be confused.</p>`,
		`<div><p>Opening paragraph that is long enough to be unique in this chapter alone.</p><p>The lantern swung over the harbour while the old captain counted every rope twice before dawn.</p></div>`)
	store := catalog.NewFilesystemStore(root, filepath.Join(t.TempDir(), "cache.json"), catalog.FilesystemOptions{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	handler, err := NewMulti(MultiConfig{AllowOpen: true, ReadingState: filepath.Join(t.TempDir(), "reading.json")},
		[]Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	save := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/reading/books/"+book.ID, strings.NewReader(body)))
		return recorder
	}
	first := save(`{"fraction":0.6,"base_revision":0,"device_id":"phone","web_anchor":{"version":1,"prefix":"","exact":"","suffix":"","spine_index":1,"progress":0.5},"koreader_locator":"/body/DocFragment[2]/body/div/p[2]/text().0"}`)
	var saved struct {
		WebAnchor *struct {
			Exact string `json:"exact"`
		} `json:"web_anchor"`
		Revision int64 `json:"revision"`
	}
	if first.Code != http.StatusOK || json.Unmarshal(first.Body.Bytes(), &saved) != nil || saved.WebAnchor == nil || !strings.HasPrefix(saved.WebAnchor.Exact, "The lantern swung") {
		t.Fatalf("the quote was not derived: %d %s", first.Code, first.Body.String())
	}
	own := save(fmt.Sprintf(`{"fraction":0.7,"base_revision":%d,"device_id":"phone","web_anchor":{"version":2,"prefix":"","exact":"A quote the phone supplied itself, long enough to count","suffix":"","spine_index":1,"progress":0.7},"koreader_locator":"/body/DocFragment[2]/body/div/p[2]/text().0"}`, saved.Revision))
	if own.Code != http.StatusOK || !strings.Contains(own.Body.String(), "A quote the phone supplied itself") {
		t.Fatalf("a quote the client supplied must be kept: %d %s", own.Code, own.Body.String())
	}
}

// The Boox reports only the start of a chapter (/body/DocFragment[N].0). That
// must replace a web position that was saved earlier, not leave its old text
// anchor behind, or the web reader reopens at the old place while Home shows
// the new percentage.
func TestChapterLevelKoSyncPositionReplacesTheWebPosition(t *testing.T) {
	root := t.TempDir()
	writeTestEPUB(t, filepath.Join(root, "One.epub"),
		`<p>The first chapter has plenty of distinctive words so that its place can be found again later.</p>`,
		`<p>Chapter two says something entirely different about lanterns, harbours and the sea at night.</p>`,
		`<p>Chapter three closes the story with a calm paragraph that nothing else in the book repeats.</p>`)
	store := catalog.NewFilesystemStore(root, filepath.Join(t.TempDir(), "cache.json"), catalog.FilesystemOptions{})
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	book := store.Books()[0]
	handler, err := NewMulti(MultiConfig{AllowOpen: true,
		KoSyncEnabled: true, KoSyncDatabase: filepath.Join(t.TempDir(), "kosync.json"), KoSyncUsername: "reader", KoSyncPassword: "secret",
		ReadingState: filepath.Join(t.TempDir(), "reading.json"),
	}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: store}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	document := koSyncFilenameMD5("One.epub")
	sum := md5.Sum([]byte("secret"))
	key := hex.EncodeToString(sum[:])
	type position struct {
		Fraction  float64         `json:"fraction"`
		Revision  int64           `json:"revision"`
		WebAnchor json.RawMessage `json:"web_anchor"`
		CFI       string          `json:"cfi"`
	}
	read := func() position {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/reading/books/"+book.ID, nil))
		var got position
		if recorder.Code != http.StatusOK || json.Unmarshal(recorder.Body.Bytes(), &got) != nil {
			t.Fatalf("read = %d %s", recorder.Code, recorder.Body.String())
		}
		return got
	}

	web := `{"fraction":0.1,"base_revision":0,"device_id":"web","cfi":"epubcfi(/6/2)","web_anchor":{"version":2,"prefix":"","exact":"The first chapter has plenty of distinctive words so that its place can be found again later.","suffix":"","spine_index":0,"progress":0.1}}`
	save := httptest.NewRecorder()
	handler.ServeHTTP(save, httptest.NewRequest(http.MethodPut, "/api/reading/books/"+book.ID, strings.NewReader(web)))
	if save.Code != http.StatusOK {
		t.Fatalf("web save = %d %s", save.Code, save.Body.String())
	}
	if got := read(); len(got.WebAnchor) == 0 || string(got.WebAnchor) == "null" {
		t.Fatalf("web position lost its anchor: %+v", got)
	}

	time.Sleep(1100 * time.Millisecond)
	body := fmt.Sprintf(`{"document":%q,"progress":"/body/DocFragment[3].0","percentage":0.66,"device":"Boox","device_id":"boox-1"}`, document)
	put := httptest.NewRecorder()
	handler.ServeHTTP(put, koSyncRequest(http.MethodPut, "/syncs/progress", "reader", key, body))
	if put.Code != http.StatusOK {
		t.Fatalf("progress put = %d %s", put.Code, put.Body.String())
	}
	got := read()
	if got.Fraction < 0.65 || got.Fraction > 0.67 {
		t.Fatalf("KOReader percentage was not taken: %+v", got)
	}
	if got.CFI != "" {
		t.Fatalf("the old web CFI survived a KOReader move: %q", got.CFI)
	}
	if len(got.WebAnchor) != 0 && string(got.WebAnchor) != "null" && strings.Contains(string(got.WebAnchor), "The first chapter") {
		t.Fatalf("the old web anchor survived a KOReader move: %s", got.WebAnchor)
	}
}
