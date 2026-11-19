package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
	"mochil/internal/customcolumns"
	"mochil/internal/reading"
)

// journalTransferServer is the usual transfer fixture with a journal file of its
// own, so a test can see exactly what the server recorded.
func journalTransferServer(t *testing.T) (*multiServer, *transferJournal, string, string) {
	t.Helper()
	server, one, two := transferServer(t)
	server.cfg.TransferJournalState = filepath.Join(t.TempDir(), "library-transfers.json")
	journal, err := server.transfers()
	if err != nil {
		t.Fatal(err)
	}
	return server, journal, one, two
}

// brokenReadingServer makes reading.Store unable to save: its state file is
// replaced by a directory, so the atomic rename at the end of every write
// fails. Every step before reading succeeds, which is the exact shape of the
// silent partial success this journal exists to end.
func brokenReadingServer(t *testing.T) (*multiServer, *transferJournal, string, string, string) {
	t.Helper()
	server, journal, one, two := journalTransferServer(t)
	path := filepath.Join(t.TempDir(), "reading.json")
	store, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put("default", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .4}, 0); err != nil {
		t.Fatal(err)
	}
	// The save is coalesced; the file only exists once it is flushed.
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	server.reading = store
	return server, journal, one, two, path
}

func moveTwoBooks(server *multiServer, mode string) *httptest.ResponseRecorder {
	return editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1","n1"],"mode":"`+mode+`"}`)
}

func TestTransferWithAFailingStepIsRecordedAndAnswered409(t *testing.T) {
	server, journal, one, two, _ := brokenReadingServer(t)
	response := moveTwoBooks(server, "move")
	if response.Code != http.StatusConflict {
		t.Fatalf("a failed step must not answer success: %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"failed":"reading"`) || !strings.Contains(body, "will be completed automatically") {
		t.Fatalf("the answer must name the failed step and say the transfer is recorded: %s", body)
	}
	if strings.Contains(body, `"token"`) {
		t.Fatalf("no undo token can exist for a transfer that did not finish: %s", body)
	}
	if present(one, "n.epub") || !present(two, "n.epub") {
		t.Fatal("the files move before the bookkeeping is recorded")
	}
	open := journal.open()
	if len(open) != 1 {
		t.Fatalf("one open journal entry = %+v", open)
	}
	entry := open[0]
	if entry.From != "books" || entry.To != "shelf" || entry.Mode != "move" || entry.Books["a1"] != catalog.BookID("Dune/Dune 1.epub") || len(entry.Paths) != 2 {
		t.Fatalf("the entry must describe the transfer: %+v", entry)
	}
	if !entry.done(transferStepFiles) || !entry.done(transferStepMetadata) || entry.done(transferStepReading) || entry.Failed != transferStepReading {
		t.Fatalf("the entry must name the steps that finished and the one that failed: %+v", entry)
	}
	// The reading place is still keyed to the old library: that is the orphaning
	// the journal now guarantees will be repaired.
	if position, ok := server.reading.Get("default", "books", "a1"); !ok || position.Fraction != .4 {
		t.Fatalf("the failed step must leave the store as it was: %+v %v", position, ok)
	}
}

func TestTransferClearsTheJournalEntryWhenEveryStepSucceeds(t *testing.T) {
	server, journal, _, _ := journalTransferServer(t)
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1"],"mode":"move"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("move = %d %s", response.Code, response.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"done", "mode", "to", "token", "rescan"} {
		if _, ok := result[key]; !ok {
			t.Fatalf("the success answer must keep its shape, %q is missing: %s", key, response.Body.String())
		}
	}
	if result["done"] != float64(1) || result["token"] == "" || result["mode"] != "move" || result["to"] != "Shelf" {
		t.Fatalf("success = %s", response.Body.String())
	}
	if open := journal.open(); len(open) != 0 {
		t.Fatalf("a finished transfer must leave no entry: %+v", open)
	}
	raw, err := os.ReadFile(server.cfg.TransferJournalState)
	if err != nil {
		t.Fatal(err)
	}
	var persisted transferJournalData
	if err := json.Unmarshal(raw, &persisted); err != nil || len(persisted.Transfers) != 0 {
		t.Fatalf("the journal file must be empty after success: %s (%v)", raw, err)
	}
}

func TestReconcileTransfersFinishesAnOpenEntryAndIsSafeToRepeat(t *testing.T) {
	server, journal, _, _, readingPath := brokenReadingServer(t)
	if response := moveTwoBooks(server, "move"); response.Code != http.StatusConflict {
		t.Fatalf("the transfer should have been recorded: %d %s", response.Code, response.Body.String())
	}
	if err := os.Remove(readingPath); err != nil {
		t.Fatalf("the reading store should be able to save again: %v", err)
	}
	server.reconcileTransfers()
	if open := journal.open(); len(open) != 0 {
		t.Fatalf("the finished transfer must leave no entry: %+v", open)
	}
	newID := catalog.BookID("Dune/Dune 1.epub")
	if position, ok := server.reading.Get("default", "shelf", newID); !ok || position.Fraction != .4 || position.BookID != newID {
		t.Fatalf("reconcile must carry the reading place: %+v %v", position, ok)
	}
	if _, ok := server.reading.Get("default", "books", "a1"); ok {
		t.Fatal("the old reading place should be gone")
	}
	// Running it again must not need the stores at all: break the reading store
	// and check that the already-reconciled transfer is left alone.
	if err := os.Remove(readingPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(readingPath, 0o755); err != nil {
		t.Fatal(err)
	}
	server.reconcileTransfers()
	if open := journal.open(); len(open) != 0 {
		t.Fatalf("a second reconcile must not resurrect an entry: %+v", open)
	}
	if position, ok := server.reading.Get("default", "shelf", newID); !ok || position.Fraction != .4 {
		t.Fatalf("a second reconcile must not disturb a finished transfer: %+v %v", position, ok)
	}
}

func TestReconcileTransfersLeavesAnEmptyJournalAlone(t *testing.T) {
	server, journal, one, two := journalTransferServer(t)
	server.reconcileTransfers()
	server.reconcileTransfers()
	if open := journal.open(); len(open) != 0 {
		t.Fatalf("an empty journal must stay empty: %+v", open)
	}
	if !present(one, "n.epub") || present(two, "n.epub") || !present(one, "Dune/Dune 1.epub") {
		t.Fatal("reconcile with nothing recorded must not touch the libraries")
	}
	if _, err := os.Stat(server.cfg.TransferJournalState); !os.IsNotExist(err) {
		t.Fatalf("reconcile with nothing recorded must not write a journal file: %v", err)
	}
}

func TestReconcileTransfersResumesAfterTheFilesStepWithoutRepeatingIt(t *testing.T) {
	server, journal, one, two := journalTransferServer(t)
	if err := server.metadataOverrides.Apply("books", map[string]map[string]string{"n1": {"title": "Neuromancer (fixed)"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.begin(transferJournalEntry{
		From:  "books",
		To:    "shelf",
		Mode:  "move",
		Books: map[string]string{"n1": catalog.BookID("n.epub")},
		Paths: []string{"n.epub"},
	}); err != nil {
		t.Fatal(err)
	}
	// The process died after the file move but before the step was recorded.
	if err := os.Rename(filepath.Join(one, "n.epub"), filepath.Join(two, "n.epub")); err != nil {
		t.Fatal(err)
	}
	server.reconcileTransfers()
	if open := journal.open(); len(open) != 0 {
		t.Fatalf("the resumed transfer must finish: %+v", open)
	}
	if present(one, "n.epub") || !present(two, "n.epub") {
		t.Fatal("reconcile must not move an already-moved file again")
	}
	if override, ok := server.metadataOverrides.Get("shelf", catalog.BookID("n.epub")); !ok || override.Title != "Neuromancer (fixed)" {
		t.Fatalf("the metadata carry must still happen: %+v %v", override, ok)
	}
}

func TestTransferJournalRecordsCopyStepsThatCannotBeReplayed(t *testing.T) {
	server, journal, _, _ := journalTransferServer(t)
	path := filepath.Join(t.TempDir(), "columns.json")
	columns, err := customcolumns.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	column, err := columns.Add("Score", customcolumns.Number)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := columns.Set("books", map[string]map[string]string{"a1": {column.ID: "9"}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	server.customColumns = columns
	response := moveTwoBooks(server, "copy")
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"failed":"columns"`) {
		t.Fatalf("a copy whose column carry fails = %d %s", response.Code, response.Body.String())
	}
	open := journal.open()
	if len(open) != 1 {
		t.Fatalf("one open entry = %+v", open)
	}
	// A copy leaves the source books in place, so replaying the metadata and
	// column carries could overwrite later edits on the copies. The journal must
	// say so, and reconcile must leave those steps alone.
	for _, step := range []string{transferStepMetadata, transferStepColumns} {
		if open[0].Skipped[step] == "" {
			t.Fatalf("%s must be recorded as not replayable for a copy: %+v", step, open[0])
		}
	}
	if open[0].Skipped[transferStepFiles] != "" || open[0].Skipped[transferStepReading] != "" {
		t.Fatalf("only the carries that could overwrite later edits are skipped: %+v", open[0])
	}
}

// A stale journal entry that names work already done must not undo later work.
// Each step is written to be replayed: a move deletes what it carried from the
// source, so a second application finds nothing and leaves the target's newer
// values (and reading place) alone.
func TestReconcileDoesNotOverwriteWorkDoneAfterAFinishedTransfer(t *testing.T) {
	server, journal, _, _ := journalTransferServer(t)
	columns, err := customcolumns.Open(filepath.Join(t.TempDir(), "columns.json"))
	if err != nil {
		t.Fatal(err)
	}
	column, err := columns.Add("Score", customcolumns.Number)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := columns.Set("books", map[string]map[string]string{"a1": {column.ID: "9"}}); err != nil {
		t.Fatal(err)
	}
	server.customColumns = columns
	if err := server.metadataOverrides.Apply("books", map[string]map[string]string{"a1": {"title": "corrected"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.reading.Put("default", reading.Position{LibraryID: "books", BookID: "a1", Fraction: .4}, 0); err != nil {
		t.Fatal(err)
	}
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/transfer", `{"library":"books","to":"shelf","ids":["a1"],"mode":"move"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("move = %d %s", response.Code, response.Body.String())
	}
	newID := catalog.BookID("Dune/Dune 1.epub")
	// Afterwards the moved book is edited and read further in its new library.
	if err := server.metadataOverrides.Apply("shelf", map[string]map[string]string{newID: {"title": "edited later"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := columns.Set("shelf", map[string]map[string]string{newID: {column.ID: "5"}}); err != nil {
		t.Fatal(err)
	}
	position, ok := server.reading.Get("default", "shelf", newID)
	if !ok {
		t.Fatal("the move should have carried the reading place")
	}
	if _, err := server.reading.Put("default", reading.Position{LibraryID: "shelf", BookID: newID, Fraction: .8}, position.Revision); err != nil {
		t.Fatal(err)
	}
	// A crash left the same transfer recorded although every step had finished.
	if _, err := journal.begin(transferJournalEntry{
		From:  "books",
		To:    "shelf",
		Mode:  "move",
		Books: map[string]string{"a1": newID},
		Paths: []string{"Dune/Dune 1.epub"},
	}); err != nil {
		t.Fatal(err)
	}
	server.reconcileTransfers()
	if open := journal.open(); len(open) != 0 {
		t.Fatalf("the stale entry must be cleared: %+v", open)
	}
	if override, ok := server.metadataOverrides.Get("shelf", newID); !ok || override.Title != "edited later" {
		t.Fatalf("replay overwrote a later metadata edit: %+v %v", override, ok)
	}
	if values := columns.Library("shelf")[newID]; values[column.ID] != "5" {
		t.Fatalf("replay overwrote a later column edit: %v", values)
	}
	if position, ok := server.reading.Get("default", "shelf", newID); !ok || position.Fraction != .8 {
		t.Fatalf("replay overwrote later reading progress: %+v %v", position, ok)
	}
}
