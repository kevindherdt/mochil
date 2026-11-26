package web

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/filefacts"
	"mochil/internal/isbn"
)

// epubWithCopyrightPage is an EPUB whose copyright page prints an ISBN (and a later page lists another volume's).
func epubWithCopyrightPage(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	files := [][2]string{
		{"META-INF/container.xml", `<container><rootfiles><rootfile full-path="c.opf"/></rootfiles></container>`},
		{"c.opf", `<package><manifest><item id="a" href="copyright.xhtml"/><item id="b" href="end.xhtml"/></manifest><spine><itemref idref="a"/><itemref idref="b"/></spine></package>`},
		{"copyright.xhtml", `<html><body><p>ISBN 978-0-306-40615-7 (ebook)</p></body></html>`},
		{"end.xhtml", `<html><body><p>Also by this author</p><p>978-0-7432-7356-5</p></body></html>`},
	}
	for _, file := range files {
		writer, _ := archive.Create(file[0])
		_, _ = writer.Write([]byte(file[1]))
	}
	_ = archive.Close()
	return buffer.Bytes()
}

func waitForISBN(t *testing.T, server *multiServer) map[string]any {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var status map[string]any
		_ = json.Unmarshal(editorCall(server, http.MethodGet, "/api/metadata/editor/isbn?library=books", "").Body.Bytes(), &status)
		if status["status"] == "ready" || status["status"] == "failed" {
			return status
		}
	}
	t.Fatal("the ISBN search did not finish")
	return nil
}

func TestFindingISBNsReadsTheBooksAndEditsNothing(t *testing.T) {
	server := countServer(t)
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", epubBackend{staticBackend: staticBackend{books: server.libraries[0].Store.Books()}, file: epubWithCopyrightPage(t)})
	var err error
	if server.isbnFound, err = newISBNStore(t); err != nil {
		t.Fatal(err)
	}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	status := waitForISBN(t, server)
	if status["status"] != "ready" || !strings.Contains(status["message"].(string), "1 with a clear ISBN") {
		t.Fatalf("status = %v", status)
	}
	rows := listEditor(t, server, "&filter=isbn_found")
	if rows.Total != 1 || rows.Items[0].IsbnFound != "9780306406157" {
		t.Fatalf("found = %+v", rows)
	}
	if !strings.Contains(rows.Items[0].IsbnNote, "Also in the book: 9780743273565") {
		t.Fatalf("the note names the other ISBN in the book: %q", rows.Items[0].IsbnNote)
	}
	if rows.Items[0].Identifiers != "" && strings.Contains(rows.Items[0].Identifiers, "isbn:") {
		t.Fatal("finding an ISBN must not write it into the book's identifiers")
	}
	if edited := listEditor(t, server, "&filter=edited"); edited.Total != 0 {
		t.Fatal("nothing may be edited by the search")
	}
	if needs := listEditor(t, server, "&filter=issues"); needs.Counts["isbn_found"] != 1 {
		t.Fatalf("counts = %v", needs.Counts)
	}
}

func TestTheISBNSearchCanBeStopped(t *testing.T) {
	server := countServer(t)
	opening := make(chan struct{}, 1)
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", stuckBackend{staticBackend: staticBackend{books: server.libraries[0].Store.Books()}, opening: opening})
	var err error
	if server.isbnFound, err = newISBNStore(t); err != nil {
		t.Fatal(err)
	}
	editorCall(server, http.MethodPost, "/api/metadata/editor/isbn", `{"library":"books"}`)
	select {
	case <-opening:
	case <-time.After(5 * time.Second):
		t.Fatal("never started")
	}
	if stop := editorCall(server, http.MethodDelete, "/api/metadata/editor/isbn?library=books", ""); !strings.Contains(stop.Body.String(), `"stopped":true`) {
		t.Fatalf("stop: %s", stop.Body)
	}
}

func TestTheNoteSaysWhyOrThatTheCallWasTooClose(t *testing.T) {
	clear := isbn.Result{Own: "9780306406157", Candidates: []isbn.Candidate{{ISBN: "9780306406157", Reasons: []string{"on the copyright page"}}}}
	if note := isbnNote(clear); !strings.Contains(note, "on the copyright page") {
		t.Fatal(note)
	}
	close := isbn.Result{Ambiguous: true, Candidates: []isbn.Candidate{{ISBN: "9780306406157", Reasons: []string{"labelled ISBN"}}, {ISBN: "9780743273565", Reasons: []string{"labelled ISBN"}}}}
	note := isbnNote(close)
	if !strings.Contains(note, "Not certain") || !strings.Contains(note, "9780743273565") || !strings.Contains(note, "Closest: 9780306406157") {
		t.Fatal(note)
	}
	if note := isbnNote(isbn.Result{}); !strings.Contains(note, "No ISBN") {
		t.Fatal(note)
	}
}

// A book can print a dozen ISBNs — a colophon, a spine, a footer on every page — and the note is a tooltip on one
// cell. Listing every candidate with every reason made this the largest field on an editor row: measured on a 5 544-book
// library it was 233 kB of a 935 kB page, a quarter of everything sent, to say something nobody reads to the end.
func TestTheNoteStaysShortWhenABookPrintsManyISBNs(t *testing.T) {
	many := isbn.Result{}
	for i := 0; i < 40; i++ {
		many.Candidates = append(many.Candidates, isbn.Candidate{
			ISBN:    fmt.Sprintf("978000000%04d", i),
			Reasons: []string{"on the copyright page", "labelled ISBN"},
		})
	}
	note := isbnNote(many)
	if len([]rune(note)) > 320 {
		t.Fatalf("the note is %d characters, which is not a tooltip: %q", len([]rune(note)), note)
	}
	// 40 candidates: the best, two named extras, and the other 37 counted.
	if !strings.Contains(note, "and 37 more") {
		t.Fatalf("the note drops the candidates it does not list without saying how many there are: %q", note)
	}
	// The first two extras are still named: the count is not a replacement for the useful part.
	if !strings.Contains(note, "9780000000001") || !strings.Contains(note, "9780000000002") {
		t.Fatalf("the note lost the candidates nearest the top: %q", note)
	}
}

func newISBNStore(t *testing.T) (*filefacts.Store[isbn.Result], error) {
	t.Helper()
	return filefacts.Open[isbn.Result](filepath.Join(t.TempDir(), "isbn-found.json"))
}

// The note is worded only for the rows a response carries, after the whole library has been sorted, so a row that
// lands on a later page must still get its own book's note and not one left over from the row built last.
func TestTheISBNNoteReachesTheRowOnWhicheverPageItLands(t *testing.T) {
	server := editorTestServer(t)
	store, err := newISBNStore(t)
	if err != nil {
		t.Fatal(err)
	}
	server.isbnFound = store
	// a1 sorts first and n1 (no series, no author) last in the editor's default order.
	reasons := map[string]string{"a1": "on the copyright page", "n1": "listed in the book's own metadata"}
	for _, book := range server.libraries[0].Store.Books() {
		reason, found := reasons[book.ID]
		if !found {
			continue
		}
		store.Put("books", book.ID, book.Size, book.Modified, isbn.Result{
			Own:        "9780306406157",
			Candidates: []isbn.Candidate{{ISBN: "9780306406157", Reasons: []string{reason}}},
		})
	}
	first := listEditor(t, server, "&limit=1")
	if len(first.Items) != 1 || first.Items[0].ID != "a1" || !strings.Contains(first.Items[0].IsbnNote, reasons["a1"]) {
		t.Fatalf("first page = %+v", first.Items)
	}
	last := listEditor(t, server, "&limit=1&offset=4")
	if len(last.Items) != 1 || last.Items[0].ID != "n1" || !strings.Contains(last.Items[0].IsbnNote, reasons["n1"]) {
		t.Fatalf("last page = %+v", last.Items)
	}
	if strings.Contains(last.Items[0].IsbnNote, reasons["a1"]) {
		t.Fatalf("a row carries another book's note: %q", last.Items[0].IsbnNote)
	}
}

// A note that already fits is handed back as it is, and one that does not is cut on a whole character. Bytes
// outnumber characters, so counting bytes first is enough to know that most notes need no conversion at all.
func TestBoundingCountsCharactersNotBytes(t *testing.T) {
	accents := strings.Repeat("é", 12) // 24 bytes, 12 characters
	if got := bounded(20, accents); got != accents {
		t.Fatalf("a note of 12 characters was cut to %q", got)
	}
	exact := strings.Repeat("x", 20)
	if got := bounded(20, exact); got != exact {
		t.Fatalf("a note at the limit was cut to %q", got)
	}
	got := bounded(10, strings.Repeat("é", 30))
	if len([]rune(got)) != 10 || []rune(got)[8] != 'é' || !strings.HasSuffix(got, "…") {
		t.Fatalf("a long note was cut in the wrong place: %q", got)
	}
}

// A book that already carries an ISBN is skipped by default, and that is the point of the recheck option: reading it
// anyway is the only way to learn that the number in the metadata is not the number printed in the file.
func TestRecheckReadsBooksThatAlreadyHaveAnISBN(t *testing.T) {
	server := countServer(t)
	books := server.libraries[0].Store.Books()
	// Give the book whose file the fixture can serve an ISBN that the file disagrees with.
	for index := range books {
		if books[index].ID == "a1" {
			books[index].Identifiers = map[string]string{"isbn": "9781111111111"}
		}
	}
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", epubBackend{
		staticBackend: staticBackend{books: books},
		file:          epubWithCopyrightPage(t),
	})
	var err error
	if server.isbnFound, err = newISBNStore(t); err != nil {
		t.Fatal(err)
	}

	// Default run: nothing to find, so nothing is read and nothing is reported.
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	// The message wording depends on whether anything else was unreadable, so the state is what gets asserted: skipped
	// means nothing was stored about this book, so the column stays empty and no badge appears.
	if status := waitForISBN(t, server); status["status"] != "ready" {
		t.Fatalf("status = %v", status)
	}
	for _, row := range listEditor(t, server, "").Items {
		if row.ID == "a1" && row.IsbnFound != "" {
			t.Fatalf("a book with an ISBN was read without being asked: %+v", row)
		}
	}
	if rows := listEditor(t, server, "&filter=issues"); rows.Counts["isbn_differs"] != 0 {
		t.Fatalf("nothing should differ before a recheck: %v", rows.Counts)
	}

	// Recheck: the book is read, the file's ISBN is shown beside the metadata one, and the mismatch is a badge.
	if recheck := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn", `{"library":"books","recheck":true}`); recheck.Code != http.StatusAccepted {
		t.Fatalf("recheck: %d %s", recheck.Code, recheck.Body)
	}
	if status := waitForISBN(t, server); status["status"] != "ready" {
		t.Fatalf("status = %v", status)
	}
	rows := listEditor(t, server, "")
	var checked *editorRow
	for index := range rows.Items {
		if rows.Items[index].ID == "a1" {
			checked = &rows.Items[index]
		}
	}
	if checked == nil || checked.IsbnFound != "9780306406157" {
		t.Fatalf("the file's ISBN is reported for a book that already has one: %+v", checked)
	}
	if counts := listEditor(t, server, "&filter=issues").Counts; counts["isbn_differs"] != 1 {
		t.Fatalf("counts = %v", counts)
	}
	if differing := listEditor(t, server, "&filter=isbn_differs"); differing.Total != 1 {
		t.Fatalf("the differing ISBN is filterable: %+v", differing)
	}
}
