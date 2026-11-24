package web

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/enrichment"
	"mochil/internal/filefacts"
	"mochil/internal/ingest"
)

// metadataFileServer is the transfer fixture: a real folder with the books in it, and no managed ingest configured at
// all — writing a sidecar must not depend on the import machinery being set up.
func metadataFileServer(t *testing.T) (*multiServer, string) {
	t.Helper()
	server, one, _ := transferServer(t)
	return server, one
}

// The folder a person drags books into is often the library they are fixing, and managed ingest refuses to import into
// it — its roots must not overlap. Writing metadata beside the book is not an import, so it works there anyway; getting
// this wrong is not hypothetical, it is what the first live run did.
func TestMetadataFilesWorkInADropFolderThatIsAlsoALibrary(t *testing.T) {
	server, one := metadataFileServer(t)
	executor, err := ingest.NewExecutor(one, filepath.Join(t.TempDir(), "managed"), ingest.NewJournalStore(filepath.Join(t.TempDir(), "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	server.managedIngest = executor
	if _, ok := server.executorFor("books"); ok {
		t.Fatal("the fixture no longer reproduces an overlapping drop folder")
	}
	response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1"]}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"written":1`) {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if !present(one, "Dune/Dune 1.opf") {
		t.Fatal("nothing was written into the drop folder")
	}
}

func readSidecar(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

// The whole point of this file: what Mochil was told about a book has to leave Mochil. Calibre reads metadata.opf when
// it adds a folder, so the corrections made in the editor — which live in Mochil's own override store until now — have
// to be written there, from the stored fields rather than from anything the EPUB was born with.
func TestWritingMetadataFilesPutsTheSavedFieldsBesideTheBook(t *testing.T) {
	server, one := metadataFileServer(t)
	if err := server.metadataOverrides.Apply("books", map[string]map[string]string{
		"a1": {"title": "Dune (fixed)", "identifiers": "isbn:9780441172719, hardcover:644"},
	}); err != nil {
		t.Fatal(err)
	}
	response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1","n1"]}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"written":2`) {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	// The name matters as much as the content: Calibre reads "metadata.opf" from a folder that holds one book, and pairs
	// "<book>.opf" with "<book>.epub" when it holds several. Writing the wrong one is a sidecar nobody reads.
	if !present(one, "metadata.opf") || !present(one, "Dune/Dune 1.opf") {
		t.Fatal("a lone book gets metadata.opf; a folder of several pairs its sidecars by name")
	}
	sidecar := readSidecar(t, one, "Dune/Dune 1.opf")
	if !strings.Contains(sidecar, "Dune (fixed)") || !strings.Contains(sidecar, "9780441172719") || !strings.Contains(sidecar, "hardcover") {
		t.Fatalf("the sidecar does not carry what was saved: %s", sidecar)
	}
	// The book file itself is untouched: writing a sidecar must never be a way to lose an EPUB.
	if body, err := os.ReadFile(filepath.Join(one, "Dune", "Dune 1.epub")); err != nil || string(body) != "d1" {
		t.Fatalf("the book was changed: %q %v", body, err)
	}

	// A sidecar that is already there is left alone, because Calibre may have written it and we cannot know whose it is.
	again := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1","n1"]}`)
	if !strings.Contains(again.Body.String(), `"exists":2`) || !strings.Contains(again.Body.String(), "left alone") {
		t.Fatalf("a second write must not overwrite: %s", again.Body.String())
	}

	if err := server.metadataOverrides.Apply("books", map[string]map[string]string{"a1": {"title": "Dune (fixed again)"}}); err != nil {
		t.Fatal(err)
	}
	over := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1"],"overwrite":true}`)
	if !strings.Contains(over.Body.String(), `"written":1`) || !strings.Contains(readSidecar(t, one, "Dune/Dune 1.opf"), "Dune (fixed again)") {
		t.Fatalf("overwrite = %s", over.Body.String())
	}
}

// Calibre owns its own metadata.opf inside its own book folders. Writing ours over it would be the second-writer
// problem in the one place it is least forgivable, so the editor refuses and says why.
func TestWritingMetadataFilesRefusesACalibreLibrary(t *testing.T) {
	server, one := metadataFileServer(t)
	server.libraries[0].SourceType = "calibre"
	response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1"]}`)
	if response.Code != 409 || !strings.Contains(response.Body.String(), "Calibre") {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if present(one, "Dune/metadata.opf") {
		t.Fatal("a Calibre library's folder was written to")
	}
}

func buildTestEPUB(t *testing.T, path string) {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	stored, _ := archive.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	stored.Write([]byte("application/epub+zip"))
	container, _ := archive.Create("META-INF/container.xml")
	container.Write([]byte(`<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="c.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))
	pkg, _ := archive.Create("c.opf")
	pkg.Write([]byte(`<package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Old</dc:title></metadata></package>`))
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// "Embed" is what makes the fix survive a drag of the EPUB alone, with no sidecar beside it. A book that cannot be
// embedded keeps its sidecar and is reported failed, because embedding was asked for and did not happen.
func TestWritingMetadataFilesCanEmbedIntoTheEPUBAndSaysWhenItCannot(t *testing.T) {
	server, one := metadataFileServer(t)
	buildTestEPUB(t, filepath.Join(one, "Dune", "Dune 1.epub"))
	if err := server.metadataOverrides.Apply("books", map[string]map[string]string{"a1": {"title": "Dune (embedded)"}}); err != nil {
		t.Fatal(err)
	}
	response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1","a2"],"embed":true}`)
	if response.Code != 200 {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if got, err := ingest.ReadEmbedded(filepath.Join(one, "Dune", "Dune 1.epub")); err != nil || got.Title != "Dune (embedded)" {
		t.Fatalf("the EPUB was not embedded: %+v %v", got, err)
	}
	body := response.Body.String()
	// a2 is not a real EPUB in this fixture, so it cannot be embedded: it must say so rather than claim success.
	if !strings.Contains(body, `"written":1`) || !strings.Contains(body, `"failed":1`) || !strings.Contains(body, "not a valid zip") {
		t.Fatalf("%s", body)
	}
	if !present(one, "Dune/Dune 1.epub") || !present(one, "Dune/Dune 2.epub") {
		t.Fatal("a book disappeared")
	}
}

// Embedding rewrites the book, and a finding is filed under the book's size and modification time — so on the next
// load the match would be gone, exactly when it had just been used. It is re-filed under the file as it now is.
func TestEmbeddingKeepsTheMatchFindingOnTheRewrittenFile(t *testing.T) {
	server, one := metadataFileServer(t)
	path := filepath.Join(one, "Dune", "Dune 1.epub")
	buildTestEPUB(t, path)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	store, err := filefacts.Open[bulkMatch](filepath.Join(t.TempDir(), "found.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The catalogue's idea of the book has to be the file's real facts, which is what a scan produces.
	books := server.libraries[0].Store.Books()
	for i := range books {
		if books[i].ID == "a1" {
			books[i].Size, books[i].Modified = before.Size(), before.ModTime()
		}
	}
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", staticBackend{books: books})
	position := 1.0
	store.Put("books", "a1", before.Size(), before.ModTime(), bulkMatch{
		Version: matchFindingVersion, How: "isbn", Clear: true,
		Candidates: []matchCandidate{{Suggestion: enrichment.Suggestion{Title: "Dune", ProviderID: "5", Series: "Dune", SeriesIndex: &position}, Score: 1}},
	})
	server.matchFound = store
	response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":["a1"],"embed":true}`)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"written":1`) {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() == before.Size() && after.ModTime().Equal(before.ModTime()) {
		t.Fatal("the fixture did not actually rewrite the file, so nothing is proven")
	}
	found, ok := store.Get("books", "a1", after.Size(), after.ModTime())
	if !ok || len(found.Candidates) == 0 || found.Candidates[0].Series != "Dune" {
		t.Fatalf("the finding did not follow the rewritten file: %+v %v", found, ok)
	}
}

func TestWritingMetadataFilesRejectsAnEmptyOrOversizedBatch(t *testing.T) {
	server, _ := metadataFileServer(t)
	if response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":[]}`); response.Code != 400 {
		t.Fatalf("empty = %d", response.Code)
	}
	many := make([]string, editorMaximumSidecars+1)
	for i := range many {
		many[i] = `"x"`
	}
	if response := editorCall(server, "POST", "/api/metadata/editor/metadata-files", `{"library":"books","ids":[`+strings.Join(many, ",")+`]}`); response.Code != 400 {
		t.Fatalf("oversized = %d", response.Code)
	}
	if _, ok := server.library("books"); !ok {
		t.Fatal("fixture")
	}
	var _ = catalog.Book{}
}
