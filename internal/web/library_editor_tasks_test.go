package web

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/filefacts"
	"mochil/internal/isbn"
)

// taskServer is the editor's test library with the three things the tasks panel reads: the ISBN findings,
// the Hardcover findings and a dismissal store with a real file behind it. It starts from countServer
// because the panel also reads the background jobs, and those only exist there.
func taskServer(t *testing.T) *multiServer {
	t.Helper()
	server := countServer(t)
	var err error
	if server.isbnFound, err = newISBNStore(t); err != nil {
		t.Fatal(err)
	}
	if server.matchFound, err = filefacts.Open[bulkMatch](filepath.Join(t.TempDir(), "hardcover-found.json")); err != nil {
		t.Fatal(err)
	}
	if server.taskDismissals, err = openTaskDismissals(filepath.Join(t.TempDir(), "task-dismissals.json")); err != nil {
		t.Fatal(err)
	}
	return server
}

// seedTaskResults gives two books a found ISBN (a1, s1) and two a match (n1, s1). a2 has only an unsure
// ISBN reading, which is a finding the editor cannot fill in, and nothing else has a result at all.
func seedTaskResults(t *testing.T, server *multiServer) {
	t.Helper()
	for _, book := range server.libraries[0].Store.Books() {
		switch book.ID {
		case "a1", "s1":
			server.isbnFound.Put("books", book.ID, book.Size, book.Modified, isbn.Result{Own: "9780306406157"})
		case "a2":
			server.isbnFound.Put("books", book.ID, book.Size, book.Modified, isbn.Result{Ambiguous: true, Candidates: []isbn.Candidate{{ISBN: "9780306406157"}}})
		}
		switch book.ID {
		case "n1", "s1":
			server.matchFound.Put("books", book.ID, book.Size, book.Modified, bulkMatch{Clear: true, Candidates: []matchCandidate{{Score: 1}}})
		}
	}
}

func listTasks(t *testing.T, server *multiServer, library string) []editorTask {
	t.Helper()
	response := editorCall(server, http.MethodGet, "/api/metadata/editor/tasks?library="+library, "")
	if response.Code != http.StatusOK {
		t.Fatalf("tasks = %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Tasks []editorTask `json:"tasks"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("tasks are not readable JSON: %v %s", err, response.Body.String())
	}
	return payload.Tasks
}

func tasksByTool(t *testing.T, server *multiServer, library string) map[string]editorTask {
	t.Helper()
	rows := map[string]editorTask{}
	for _, task := range listTasks(t, server, library) {
		rows[task.Tool] = task
	}
	return rows
}

// The panel is a list of what the tools could do here, so every tool has a row whether it has run or not.
// A tool that is missing reads as a tool that does not exist.
func TestThePanelAlwaysListsEveryToolInOrder(t *testing.T) {
	server := taskServer(t)
	seedTaskResults(t, server)
	tasks := listTasks(t, server, "books")
	want := []struct {
		tool  string
		title string
		found int
	}{
		{editorToolISBN, "Find ISBNs", 2},
		{editorToolMatch, "Hardcover matches", 2},
		{editorToolTidy, "Clean up titles", 0},
	}
	if len(tasks) != len(want) {
		t.Fatalf("the panel lists %d tools, not %d: %+v", len(tasks), len(want), tasks)
	}
	for index, expected := range want {
		got := tasks[index]
		if got.Tool != expected.tool || got.Title != expected.title || got.Found != expected.found {
			t.Errorf("row %d = %+v, want tool %q titled %q with %d found", index, got, expected.tool, expected.title, expected.found)
		}
		if got.State != editorTaskIdle || got.RanAt != nil || got.Dismissed != 0 {
			t.Errorf("a tool that has not run reads as %+v", got)
		}
	}
	// The keys are the panel's contract, and a tool that never ran carries no run time at all rather than a
	// zero one it would have to know to ignore.
	raw := editorCall(server, http.MethodGet, "/api/metadata/editor/tasks?library=books", "").Body.String()
	for _, want := range []string{`"tool":"isbn"`, `"title":"Find ISBNs"`, `"state":"idle"`, `"found":2`, `"dismissed":0`} {
		if !strings.Contains(raw, want) {
			t.Errorf("the payload is missing %s: %s", want, raw)
		}
	}
	if strings.Contains(raw, "ran_at") {
		t.Errorf("a tool that never ran carries a run time: %s", raw)
	}
}

// Discarding one tool's results hides them from that tool only. This is the bug the per-tool store exists
// for: the per-book list hid the same book from the ISBNs, the matches and the cleanup at once.
func TestDiscardingOneToolsResultsLeavesTheOtherToolsAlone(t *testing.T) {
	for _, test := range []struct {
		tool string
		book string
	}{
		{editorToolISBN, "a1"},
		{editorToolMatch, "n1"},
		{editorToolTidy, "n1"},
	} {
		t.Run(test.tool, func(t *testing.T) {
			server := taskServer(t)
			seedTaskResults(t, server)
			before := tasksByTool(t, server, "books")
			body := `{"library":"books","books":["` + test.book + `"]}`
			response := editorCall(server, http.MethodPost, "/api/metadata/editor/tasks/"+test.tool+"/dismiss", body)
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Fatalf("dismiss = %d %q, want 204 with no body", response.Code, response.Body.String())
			}
			after := tasksByTool(t, server, "books")
			if after[test.tool].Dismissed != 1 {
				t.Fatalf("dismissed = %d, want 1", after[test.tool].Dismissed)
			}
			if after[test.tool].Found != before[test.tool].Found {
				t.Fatalf("found changed from %d to %d: a discard must not change what was found", before[test.tool].Found, after[test.tool].Found)
			}
			for _, other := range editorTaskOrder {
				if other == test.tool {
					continue
				}
				if after[other].Dismissed != 0 {
					t.Fatalf("discarding for %s also discarded %d for %s", test.tool, after[other].Dismissed, other)
				}
			}
			// It is in the file, not only in memory: a restart is what the store is for.
			reopened, err := openTaskDismissals(server.taskDismissals.path)
			if err != nil {
				t.Fatal(err)
			}
			if reopened.count("books", test.tool) != 1 {
				t.Fatalf("the dismissal was not written: %d on disk", reopened.count("books", test.tool))
			}
		})
	}
}

// An empty list means "everything this tool found", filled from the tool's own results — not every book in
// the library, which would hide results the tool never had.
func TestDismissingEverythingTakesOnlyWhatTheToolFound(t *testing.T) {
	for _, tool := range []string{editorToolISBN, editorToolMatch} {
		t.Run(tool, func(t *testing.T) {
			server := taskServer(t)
			seedTaskResults(t, server)
			found := tasksByTool(t, server, "books")[tool].Found
			response := editorCall(server, http.MethodPost, "/api/metadata/editor/tasks/"+tool+"/dismiss", `{"library":"books","books":[]}`)
			if response.Code != http.StatusNoContent {
				t.Fatalf("dismiss all = %d %s", response.Code, response.Body.String())
			}
			task := tasksByTool(t, server, "books")[tool]
			if task.Dismissed != found {
				t.Fatalf("dismissed = %d, want the %d that were found", task.Dismissed, found)
			}
			if task.Dismissed == len(server.libraries[0].Store.Books()) {
				t.Fatal("everything in the library was dismissed, not just what the tool found")
			}
		})
	}
}

// A discard that could not be undone would be a trap, so clearing the set is part of the contract.
func TestClearingADiscardBringsTheResultsBack(t *testing.T) {
	for _, tool := range editorTaskOrder {
		t.Run(tool, func(t *testing.T) {
			server := taskServer(t)
			seedTaskResults(t, server)
			found := tasksByTool(t, server, "books")[tool].Found
			if response := editorCall(server, http.MethodPost, "/api/metadata/editor/tasks/"+tool+"/dismiss", `{"library":"books","books":["a1","n1","s1"]}`); response.Code != http.StatusNoContent {
				t.Fatalf("dismiss = %d %s", response.Code, response.Body.String())
			}
			if got := tasksByTool(t, server, "books")[tool].Dismissed; got != 3 {
				t.Fatalf("dismissed = %d, want 3", got)
			}
			response := editorCall(server, http.MethodDelete, "/api/metadata/editor/tasks/"+tool+"/dismiss?library=books", "")
			if response.Code != http.StatusNoContent {
				t.Fatalf("clear = %d %s", response.Code, response.Body.String())
			}
			task := tasksByTool(t, server, "books")[tool]
			if task.Dismissed != 0 {
				t.Fatalf("dismissed = %d after clearing", task.Dismissed)
			}
			if task.Found != found {
				t.Fatalf("clearing a discard changed what was found from %d to %d", found, task.Found)
			}
		})
	}
}

// The tool name is part of the path, so a name that is not one of the three has to be refused before
// anything is read or written; a plain 400 says which part was wrong.
func TestAnUnknownToolIsRefused(t *testing.T) {
	for _, path := range []string{
		"/api/metadata/editor/tasks/ask/dismiss",
		"/api/metadata/editor/tasks/ISBN/dismiss",
		"/api/metadata/editor/tasks/isbnx/dismiss",
	} {
		t.Run(path, func(t *testing.T) {
			server := taskServer(t)
			response := editorCall(server, http.MethodPost, path, `{"library":"books","books":["a1"]}`)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("unknown tool = %d %s, want 400", response.Code, response.Body.String())
			}
			if server.taskDismissals.count("books", editorToolISBN) != 0 {
				t.Fatal("a refused request wrote something")
			}
		})
	}
}

// Library ids and book ids are not unique across libraries: the same book can be in a books library and in
// a comics library, and discarding it in one must not discard it in the other.
func TestDismissalsAreKeptPerLibrary(t *testing.T) {
	server := taskServer(t)
	seedTaskResults(t, server)
	server.libraries = append(server.libraries, Library{
		ID: "comics", Name: "Comics", Kind: "books",
		Store: server.metadataOverrides.Wrap("comics", staticBackend{books: server.libraries[0].Store.Books()}),
	})
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/tasks/isbn/dismiss", `{"library":"books","books":["a1"]}`); response.Code != http.StatusNoContent {
		t.Fatalf("dismiss = %d %s", response.Code, response.Body.String())
	}
	if got := tasksByTool(t, server, "books")[editorToolISBN].Dismissed; got != 1 {
		t.Fatalf("books dismissed = %d, want 1", got)
	}
	if got := tasksByTool(t, server, "comics")[editorToolISBN].Dismissed; got != 0 {
		t.Fatalf("the same book id was discarded in comics too: %d", got)
	}
	if got := server.taskDismissals.count("comics", editorToolISBN); got != 0 {
		t.Fatalf("the store holds %d for comics", got)
	}
}

// A run under way reads as running, and once it ends the row says when. Stopping it counts as ending: the
// work is over and what it found is kept.
func TestTheRowSaysWhenTheRunIsUnderWayAndWhenItEnded(t *testing.T) {
	server := taskServer(t)
	opening := make(chan struct{}, 1)
	server.libraries[0].Store = server.metadataOverrides.Wrap("books", stuckBackend{staticBackend: staticBackend{books: server.libraries[0].Store.Books()}, opening: opening})
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/isbn", `{"library":"books"}`); response.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", response.Code, response.Body)
	}
	select {
	case <-opening:
	case <-time.After(5 * time.Second):
		t.Fatal("the ISBN search never started")
	}
	running := tasksByTool(t, server, "books")[editorToolISBN]
	if running.State != editorTaskRunning || running.RanAt != nil {
		t.Fatalf("a run in progress reads as %+v", running)
	}
	editorCall(server, http.MethodDelete, "/api/metadata/editor/isbn?library=books", "")
	finished := tasksByTool(t, server, "books")[editorToolISBN]
	if finished.State != editorTaskFinished || finished.RanAt == nil {
		t.Fatalf("an ended run reads as %+v", finished)
	}
	if tasksByTool(t, server, "books")[editorToolMatch].State != editorTaskIdle {
		t.Fatal("the ISBN search gave the matches a state of their own")
	}
}

// An answer that has been written is not work left to do. Staging skips a book that already carries the value, so a
// count that ignored it never fell: the row kept offering 1080 ISBNs after every one of them had been applied.
func TestAResultThatHasBeenWrittenStopsCountingAsWorkLeft(t *testing.T) {
	server := taskServer(t)
	seedTaskResults(t, server)
	if found := tasksByTool(t, server, "books")["isbn"].Found; found != 2 {
		t.Fatalf("isbn found = %d before any edit, want 2", found)
	}

	// The identifier the panel would write, written the way the panel writes it.
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit",
		`{"library":"books","changes":{"a1":{"identifiers":"isbn:9780306406157"}}}`)
	if response.Code != http.StatusOK {
		t.Fatalf("edit = %d %s", response.Code, response.Body.String())
	}

	if found := tasksByTool(t, server, "books")["isbn"].Found; found != 1 {
		t.Fatalf("isbn found = %d after the ISBN was written, want 1", found)
	}
}

func TestAnIdentifierListIsReadByScheme(t *testing.T) {
	for _, test := range []struct {
		identifiers map[string]string
		scheme      string
		want        bool
	}{
		{map[string]string{"isbn": "9780306406157"}, "isbn", true},
		{map[string]string{"isbn": "  "}, "isbn", false},
		{map[string]string{"hardcover": "974285"}, "isbn", false},
		{map[string]string{}, "isbn", false},
	} {
		if got := identifierHas(test.identifiers, test.scheme); got != test.want {
			t.Errorf("identifierHas(%v, %q) = %v, want %v", test.identifiers, test.scheme, got, test.want)
		}
	}
}
