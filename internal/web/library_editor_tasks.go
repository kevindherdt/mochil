package web

import (
	"net/http"
	"strings"
	"time"
)

// The editor's tools, named as the tasks panel and its endpoints name them.
const (
	editorToolISBN  = "isbn"
	editorToolMatch = "match"
	editorToolTidy  = "tidy"
)

// editorTaskOrder is the order the panel lists the tools in: the two that read the books first, then the
// cleanup, which is the cheap one.
var editorTaskOrder = []string{editorToolISBN, editorToolMatch, editorToolTidy}

// What each state means, as the panel reads them.
const (
	editorTaskIdle     = "idle"     // never run for this library, or nothing to say about it
	editorTaskRunning  = "running"  // the background job is under way now
	editorTaskFinished = "finished" // it has ended at least once; ran_at says when
)

// editorTaskTitles is what each tool is called where a person reads it. The endpoint names them isbn,
// match and tidy; nobody should have to know that.
var editorTaskTitles = map[string]string{
	editorToolISBN:  "Find ISBNs",
	editorToolMatch: "Hardcover matches",
	editorToolTidy:  "Clean up titles",
}

// editorTask is one row of the Tasks panel: what the tool found for this library, how much of that was
// discarded, whether it is running and when it last ended.
type editorTask struct {
	Tool      string     `json:"tool"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Found     int        `json:"found"`
	Dismissed int        `json:"dismissed"`
	RanAt     *time.Time `json:"ran_at,omitempty"`
}

// editorTasks answers GET /api/metadata/editor/tasks?library= with one row per tool, always all of them.
//
// The panel needs a row for a tool that has never run, because "never run" is itself the answer to "why is
// there nothing here"; a list that only carried tools with results made an untouched tool look absent.
func (m *multiServer) editorTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	lib, ok := m.editorLibrary(w, r.URL.Query().Get("library"))
	if !ok {
		return
	}
	tasks := make([]editorTask, 0, len(editorTaskOrder))
	for _, tool := range editorTaskOrder {
		tasks = append(tasks, m.editorTask(tool, lib))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// editorTask builds one row. Found and dismissed are counted separately on purpose: the panel shows both,
// and a result that was discarded is still something the tool found.
func (m *multiServer) editorTask(tool string, lib Library) editorTask {
	task := editorTask{
		Tool:      tool,
		Title:     editorTaskTitles[tool],
		State:     editorTaskIdle,
		Found:     len(m.taskFoundBooks(tool, lib)),
		Dismissed: m.taskDismissals.count(lib.ID, tool),
	}
	running, finished := m.editorTaskRun(tool, lib.ID)
	switch {
	case running:
		task.State = editorTaskRunning
	case !finished.IsZero():
		task.State = editorTaskFinished
		// Seconds are what the panel prints ("run 10:41") and what the payload documents; the job's own
		// timestamp carries nanoseconds nobody reads.
		at := finished.UTC().Truncate(time.Second)
		task.RanAt = &at
	}
	return task
}

// editorTaskRun reports the tool's background job: whether it is under way now, and when it last ended.
// A queued job counts as running, because from the panel's side the work has been asked for and is coming.
func (m *multiServer) editorTaskRun(tool, library string) (running bool, finished time.Time) {
	job, hasJob := editorToolJob(tool)
	if !hasJob || m.backgroundJobs == nil {
		return false, time.Time{}
	}
	state, found := m.backgroundJobs.State(job.key(library))
	if !found {
		return false, time.Time{}
	}
	switch state.Status {
	case "queued", "running":
		return true, time.Time{}
	}
	if state.FinishedAt != nil {
		return false, *state.FinishedAt
	}
	return false, time.Time{}
}

// editorToolJob is the background job behind a tool. Clean up titles has none: its preview is computed for
// the books a request names, in the request, and nothing is kept afterwards.
func editorToolJob(tool string) (editorJob, bool) {
	switch tool {
	case editorToolISBN:
		return isbnJob, true
	case editorToolMatch:
		return matchJob, true
	}
	return editorJob{}, false
}

// taskFoundBooks lists the books a tool has a usable result for in one library, in catalogue order.
//
// It is the one place that decides what "found" means, so the count in the panel, the number dismissed by
// "discard the rest" and the rows the job wrote cannot drift apart:
//
//   - isbn counts a book whose file printed one clear ISBN **and** whose metadata does not carry one yet.
//     An unsure reading is a result, but not one the editor can fill in, so counting it would promise work
//     that staging then refuses; and a book already holding an ISBN is skipped by staging for the same
//     reason, so counting it left the row claiming 1080 results after every one of them had been applied.
//   - match counts a book with at least one candidate that is not linked yet. A linked book has its
//     hardcover identifier and staging skips it, so counting it had the same fault as the ISBN row.
//   - tidy has no stored results. Its proposals belong to the set of books the person selected, and are
//     computed by POST /api/metadata/editor/tidy when the panel asks, so there is nothing here to count and
//     no cheap helper to borrow; the row reports idle and 0 found rather than guessing at a number that
//     would be wrong for a different selection.
func (m *multiServer) taskFoundBooks(tool string, lib Library) []string {
	if lib.Store == nil {
		return nil
	}
	// Which store this tool reads, decided once. The default is the cleanup and any name that is not a
	// tool: nothing is stored for them, so there is nothing to count.
	switch tool {
	case editorToolISBN:
		if m.isbnFound == nil {
			return nil
		}
	case editorToolMatch:
		if m.matchFound == nil {
			return nil
		}
	default:
		return nil
	}
	var found []string
	books := publishedBooks(lib.Store)
	for i := range books {
		book := &books[i]
		switch tool {
		case editorToolISBN:
			result, ok := m.isbnFound.Get(lib.ID, book.ID, book.Size, book.Modified)
			if ok && result.Own != "" && !identifierHas(book.Identifiers, "isbn") {
				found = append(found, book.ID)
			}
		case editorToolMatch:
			match, ok := m.matchFound.Get(lib.ID, book.ID, book.Size, book.Modified)
			if ok && len(match.Candidates) > 0 && !identifierHas(book.Identifiers, "hardcover") {
				found = append(found, book.ID)
			}
		}
	}
	return found
}

// identifierHas reports whether a book's identifier list already carries a scheme ("isbn", "hardcover").
//
// It is what makes a row's count mean "still to do" rather than "ever found": both staging rules skip a book that
// already has the value, so a count that ignored it never fell, and the panel kept offering work that was finished.
func identifierHas(identifiers map[string]string, scheme string) bool {
	value, ok := identifiers[scheme]
	return ok && strings.TrimSpace(value) != ""
}

// editorTaskDismiss serves POST and DELETE on /api/metadata/editor/tasks/<tool>/dismiss.
//
// POST adds book ids to that tool's discards and answers 204; an empty list means "everything this tool
// found here", filled from taskFoundBooks so the set matches the count the panel is showing. DELETE
// empties the set, which is the way back: a discard that could not be undone would be a trap.
func (m *multiServer) editorTaskDismiss(w http.ResponseWriter, r *http.Request, tool string) {
	if _, known := editorTaskTitles[tool]; !known {
		http.Error(w, "That is not one of the editor's tools.", http.StatusBadRequest)
		return
	}
	if m.taskDismissals == nil {
		http.Error(w, "Discards cannot be saved right now.", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var request struct {
			Library string   `json:"library"`
			Books   []string `json:"books"`
		}
		if err := decodeJSONRequest(w, r, 1<<20, &request, true); err != nil {
			http.Error(w, "The request could not be read.", http.StatusBadRequest)
			return
		}
		lib, ok := m.editorLibrary(w, request.Library)
		if !ok {
			return
		}
		books := request.Books
		if len(books) == 0 {
			books = m.taskFoundBooks(tool, lib)
		}
		if err := m.taskDismissals.add(lib.ID, tool, books); err != nil {
			http.Error(w, "The discards could not be saved.", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		lib, ok := m.editorLibrary(w, r.URL.Query().Get("library"))
		if !ok {
			return
		}
		if err := m.taskDismissals.clear(lib.ID, tool); err != nil {
			http.Error(w, "The discards could not be saved.", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
