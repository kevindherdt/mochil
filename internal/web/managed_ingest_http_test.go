package web

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/ingest"
	"mochil/internal/jobs"
)

func TestManagedIngestStatusIsQuietlyDisabled(t *testing.T) {
	m := &multiServer{}
	response := httptest.NewRecorder()
	m.apiManagedIngest(response, httptest.NewRequest(http.MethodGet, "/api/metadata/managed-ingest", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"enabled":false`) {
		t.Fatalf("disabled status = %d %s", response.Code, response.Body.String())
	}
}

func TestManagedIngestPlanCanBeDiscardedBeforeCopy(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{managedIngest: executor}
	planned, err := executor.Plan([]ingest.PlanEntry{{SourcePath: "book.epub", TargetPath: "book.epub"}})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	m.apiManagedIngest(response, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/"+planned.ID+"/discard", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"state":"rolled_back"`) {
		t.Fatalf("discard = %d %s", response.Code, response.Body.String())
	}
	if _, err = os.Stat(filepath.Join(managed, "book.epub")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("discard created target: %v", err)
	}
}

func TestManagedIngestDiscoveryIsReadOnlyAndFilterable(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "Book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "Song.mp3"), []byte("song"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{managedIngest: executor}
	response := httptest.NewRecorder()
	m.apiManagedIngest(response, httptest.NewRequest(http.MethodGet, "/api/metadata/managed-ingest/discover?kind=publication&limit=1", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"source_path":"Book.epub"`) || strings.Contains(response.Body.String(), "Song.mp3") {
		t.Fatalf("discovery = %d %s", response.Code, response.Body.String())
	}
	transactions, err := executor.Transactions()
	if err != nil || len(transactions) != 0 {
		t.Fatalf("discovery mutated journal = %#v, %v", transactions, err)
	}
	if _, err = os.Stat(managed); err != nil {
		t.Fatalf("managed root missing: %v", err)
	}
}

func TestManagedIngestBatchPreviewReportsConflictsWithoutPlanning(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.epub", "two.epub"} {
		if err := os.WriteFile(filepath.Join(incoming, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{managedIngest: executor}
	request := httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/preview", strings.NewReader(`{"entries":[{"source_path":"one.epub","target_path":"Books/Same.epub"},{"source_path":"two.epub","target_path":"books/same.epub"}]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	m.apiManagedIngest(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"ready":false`) || !strings.Contains(response.Body.String(), `"code":"duplicate_target"`) {
		t.Fatalf("preview = %d %s", response.Code, response.Body.String())
	}
	transactions, err := executor.Transactions()
	if err != nil || len(transactions) != 0 {
		t.Fatalf("preview mutated journal = %#v, %v", transactions, err)
	}
}

func TestManagedIngestPlanRequiresExactConfirmationAndRunsAsJob(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	if err := os.MkdirAll(incoming, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	pool := jobs.New(1, 4)
	defer pool.Close()
	m := &multiServer{managedIngest: executor, backgroundJobs: pool}

	plan := httptest.NewRecorder()
	m.apiManagedIngest(plan, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/plan", strings.NewReader(`{"entries":[{"source_path":"book.epub","target_path":"Author/book.epub"}]}`)))
	if plan.Code != http.StatusCreated {
		t.Fatalf("plan = %d %s", plan.Code, plan.Body.String())
	}
	var transaction ingest.Transaction
	if err = json.Unmarshal(plan.Body.Bytes(), &transaction); err != nil {
		t.Fatal(err)
	}

	wrong := httptest.NewRecorder()
	m.apiManagedIngest(wrong, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/"+transaction.ID+"/execute", strings.NewReader(`{"confirmation":"wrong"}`)))
	if wrong.Code != http.StatusBadRequest {
		t.Fatalf("wrong confirmation = %d %s", wrong.Code, wrong.Body.String())
	}

	execute := httptest.NewRecorder()
	m.apiManagedIngest(execute, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/"+transaction.ID+"/execute", strings.NewReader(`{"confirmation":"`+transaction.ID+`"}`)))
	if execute.Code != http.StatusAccepted {
		t.Fatalf("execute = %d %s", execute.Code, execute.Body.String())
	}
	jobKey := "managed-ingest:execute:" + transaction.ID
	if err = pool.Wait(context.Background(), jobKey); err != nil {
		t.Fatal(err)
	}
	committed, err := executor.Transaction(transaction.ID)
	if err != nil || committed.State != ingest.TransactionCommitted {
		t.Fatalf("committed transaction = %#v, %v", committed, err)
	}
	if body, readErr := os.ReadFile(filepath.Join(managed, "Author", "book.epub")); readErr != nil || string(body) != "book" {
		t.Fatalf("managed target = %q, %v", body, readErr)
	}
}

func TestManagedIngestWritesSidecarsPerBookAndNeverReplacesThem(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	os.MkdirAll(incoming, 0o700)
	os.MkdirAll(filepath.Join(managed, "Series"), 0o755)
	os.WriteFile(filepath.Join(managed, "Series", "Series - Vol 01.epub"), []byte("book"), 0o644)
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{managedIngest: executor}
	call := func(body string) (int, string) {
		response := httptest.NewRecorder()
		m.apiManagedIngest(response, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/sidecars", strings.NewReader(body)))
		return response.Code, response.Body.String()
	}
	body := `{"entries":[{"target_path":"Series/Series - Vol 01.epub","metadata":{"title":"Vol 1","authors":["A"],"series":"Series","series_index":1,"rating":4.5,"tags":["x"]}},{"target_path":"../escape.epub","metadata":{"title":"Nope"}}]}`
	code, text := call(body)
	if code != http.StatusOK || !strings.Contains(text, `"status":"written"`) || !strings.Contains(text, `"status":"failed"`) {
		t.Fatalf("%d %s", code, text)
	}
	content, err := os.ReadFile(filepath.Join(managed, "Series", "metadata.opf"))
	if err != nil || !strings.Contains(string(content), `calibre:rating" content="9.0"`) || !strings.Contains(string(content), `calibre:series_index" content="1"`) {
		t.Fatalf("%v\n%s", err, content)
	}
	if _, text = call(strings.Replace(body, "Vol 1", "Changed", 1)); !strings.Contains(text, `"status":"exists"`) {
		t.Fatalf("a second write should leave the sidecar alone: %s", text)
	}
	if again, _ := os.ReadFile(filepath.Join(managed, "Series", "metadata.opf")); string(again) != string(content) {
		t.Fatal("the sidecar was replaced")
	}
	if _, err := os.Stat(filepath.Join(root, "escape.opf")); err == nil {
		t.Fatal("a sidecar was written outside the library")
	}
	if code, _ = call(`{"entries":[]}`); code != http.StatusBadRequest {
		t.Fatalf("no books = %d", code)
	}
	if code, _ = call(`{"entries":[{"target_path":"a.epub","metadata":{}}],"surprise":1}`); code != http.StatusBadRequest {
		t.Fatalf("unknown field = %d", code)
	}
}

func TestManagedIngestDiscoveryAcceptsOnlyKnownNamingPresets(t *testing.T) {
	root := t.TempDir()
	incoming := filepath.Join(root, "incoming")
	os.MkdirAll(incoming, 0o700)
	executor, err := ingest.NewExecutor(incoming, filepath.Join(root, "managed"), ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{managedIngest: executor}
	for naming, want := range map[string]int{"kavita": http.StatusOK, "calibre": http.StatusOK, "": http.StatusOK, "../x": http.StatusBadRequest} {
		response := httptest.NewRecorder()
		m.apiManagedIngest(response, httptest.NewRequest(http.MethodGet, "/api/metadata/managed-ingest/discover?naming="+naming, nil))
		if response.Code != want {
			t.Errorf("naming %q = %d, want %d", naming, response.Code, want)
		}
	}
}

func TestManagedIngestEmbedsMetadataIntoCopiedEPUBsOnly(t *testing.T) {
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	os.MkdirAll(incoming, 0o700)
	os.MkdirAll(managed, 0o755)
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	stored, _ := archive.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	stored.Write([]byte("application/epub+zip"))
	container, _ := archive.Create("META-INF/container.xml")
	container.Write([]byte(`<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="c.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))
	pkg, _ := archive.Create("c.opf")
	pkg.Write([]byte(`<package xmlns="http://www.idpf.org/2007/opf" version="2.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Old</dc:title></metadata></package>`))
	archive.Close()
	os.WriteFile(filepath.Join(managed, "book.epub"), buffer.Bytes(), 0o644)
	os.WriteFile(filepath.Join(managed, "scan.pdf"), []byte("pdf"), 0o644)
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{managedIngest: executor}
	response := httptest.NewRecorder()
	m.apiManagedIngest(response, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/embed", strings.NewReader(`{"entries":[{"target_path":"book.epub","metadata":{"title":"New","authors":["A"]}},{"target_path":"scan.pdf","metadata":{"title":"X"}}]}`)))
	text := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(text, `"status":"embedded"`) || !strings.Contains(text, `"status":"failed"`) {
		t.Fatalf("%d %s", response.Code, text)
	}
	if got, err := ingest.ReadEmbedded(filepath.Join(managed, "book.epub")); err != nil || got.Title != "New" || len(got.Authors) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	if pdf, _ := os.ReadFile(filepath.Join(managed, "scan.pdf")); string(pdf) != "pdf" {
		t.Fatal("a PDF was changed")
	}
	response = httptest.NewRecorder()
	m.apiManagedIngest(response, httptest.NewRequest(http.MethodPost, "/api/metadata/managed-ingest/embed", strings.NewReader(`{"entries":[]}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("no books = %d", response.Code)
	}
}

func TestImportCopiesIntoTheChosenLibraryWithItsOwnJournal(t *testing.T) {
	root := t.TempDir()
	incoming, managed, shelf := filepath.Join(root, "incoming"), filepath.Join(root, "managed"), filepath.Join(root, "shelf")
	os.MkdirAll(incoming, 0o700)
	os.MkdirAll(shelf, 0o755)
	os.WriteFile(filepath.Join(incoming, "book.epub"), []byte("book"), 0o600)
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	pool := jobs.New(1, 4)
	defer pool.Close()
	m := &multiServer{managedIngest: executor, backgroundJobs: pool, libraries: []Library{
		{ID: "shelf", Name: "Shelf", Kind: "books", SourceType: "filesystem", SourcePath: shelf},
		{ID: "cal", Name: "Calibre", Kind: "books", SourceType: "calibre", SourcePath: shelf},
		{ID: "inside-drop", Name: "Bad", Kind: "books", SourceType: "filesystem", SourcePath: incoming},
	}}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		m.apiManagedIngest(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	entries := `{"entries":[{"source_path":"book.epub","target_path":"Author/book.epub"}]}`
	for name, destination := range map[string]string{"calibre library": "cal", "unknown": "nope", "library inside the drop folder": "inside-drop"} {
		if r := call(http.MethodPost, "/api/metadata/managed-ingest/plan?destination="+destination, entries); r.Code != http.StatusNotFound {
			t.Errorf("%s = %d", name, r.Code)
		}
	}
	plan := call(http.MethodPost, "/api/metadata/managed-ingest/plan?destination=shelf", entries)
	var transaction ingest.Transaction
	if err := json.Unmarshal(plan.Body.Bytes(), &transaction); err != nil || plan.Code != http.StatusCreated {
		t.Fatalf("plan = %d %s", plan.Code, plan.Body.String())
	}
	if _, err := os.Stat(filepath.Join(managed, "Author", "book.epub")); err == nil {
		t.Fatal("nothing may be copied before the confirmation")
	}
	if lookup := call(http.MethodGet, "/api/metadata/managed-ingest/"+transaction.ID, ""); lookup.Code != http.StatusOK {
		t.Fatalf("a transaction for another destination must be found by id: %d", lookup.Code)
	}
	if run := call(http.MethodPost, "/api/metadata/managed-ingest/"+transaction.ID+"/execute", `{"confirmation":"`+transaction.ID+`"}`); run.Code != http.StatusAccepted {
		t.Fatalf("execute = %d %s", run.Code, run.Body.String())
	}
	if err := pool.Wait(context.Background(), "managed-ingest:execute:"+transaction.ID); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(shelf, "Author", "book.epub")); err != nil || string(body) != "book" {
		t.Fatalf("the book must land in the library's own folder: %q %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(managed, "Author", "book.epub")); err == nil {
		t.Fatal("the managed folder must not get a copy of a book sent to another library")
	}
	if mine, _ := executor.Transactions(); len(mine) != 0 {
		t.Fatalf("the managed folder's journal must not hold another library's transaction: %v", mine)
	}
	sidecar := call(http.MethodPost, "/api/metadata/managed-ingest/sidecars?destination=shelf", `{"entries":[{"target_path":"Author/book.epub","metadata":{"title":"Book"}}]}`)
	if !strings.Contains(sidecar.Body.String(), `"status":"written"`) {
		t.Fatalf("%s", sidecar.Body.String())
	}
	if _, err := os.Stat(filepath.Join(shelf, "Author", "metadata.opf")); err != nil {
		t.Fatal("the sidecar belongs beside the book in the library")
	}
}
