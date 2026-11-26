package web

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/customcolumns"
)

const (
	editorMaximumBody    = 1 << 20
	editorMaximumPage    = 1000
	editorOneShotMaximum = 6000
	editorMaximumEdits   = 2000
)

type editorRow struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Authors     []string `json:"authors"`
	Series      string   `json:"series"`
	SeriesIndex *float64 `json:"series_index"`
	Format      string   `json:"format"`
	Tags        []string `json:"tags"`
	Publisher   string   `json:"publisher"`
	Published   string   `json:"published"`
	Language    string   `json:"language"`
	Rating      float64  `json:"rating"` // stars, 0 to 5
	Description string   `json:"description"`
	Identifiers string   `json:"identifiers"` // "isbn:978..., hardcover:dune"
	TitleSort   string   `json:"title_sort"`
	AuthorSort  string   `json:"author_sort"`
	Path        string   `json:"path"`
	Edited      bool     `json:"edited"`
	Issues      []string `json:"issues,omitempty"`
	// Custom holds this book's values in the added columns; Progress is the signed-in person's reading, 0 to 1.
	Custom   map[string]string `json:"custom,omitempty"`
	Progress float64           `json:"progress"`
	// Words and Pages come from counting the book (see library_editor_count.go); 0 means not counted.
	Words int `json:"words"`
	Pages int `json:"pages"`
	// IsbnFound is the ISBN read out of the book itself (Tools > Find ISBNs), IsbnNote says why; both are empty if not searched.
	IsbnFound string `json:"isbn_found"`
	IsbnNote  string `json:"isbn_note"`
	// Match is the best Hardcover book found for this one (Tools > Find Hardcover matches); MatchClear says it is certain.
	Match      *matchCandidate `json:"match,omitempty"`
	MatchClear bool            `json:"match_clear"`
	MatchNote  string          `json:"match_note"`
	// Collections names the collections this book is in.
	Collections []string `json:"collections"`
	// Hardcover is where the book stands with the Hardcover sync (see hardcoverFacts.state); the note says why.
	Hardcover     string `json:"hardcover"`
	HardcoverNote string `json:"hardcover_note"`
	// book is the catalogue entry this row was built from, kept so that a field only the page being sent needs can
	// be filled after the whole library has been sorted (see addFindingNote). It is never sent: encoding/json does
	// not look at unexported fields.
	book *catalog.Book
}

// editorMatches applies the list's filter and its search to one row. The search is prepared once per request — see
// editorSearch — because folding the query and its four fields per book was three allocations per row.
func editorMatches(row editorRow, filter string, search *editorSearch) bool {
	switch filter {
	case "", "all":
	case "issues":
		var real bool
		for _, issue := range row.Issues {
			real = real || (issue != "side_story" && !qualityIssues[issue])
		}
		if !real {
			return false
		}
	case "edited":
		if !row.Edited {
			return false
		}
	default:
		found := false
		for _, issue := range row.Issues {
			found = found || issue == filter
		}
		if !found {
			return false
		}
	}
	if search == nil {
		return true
	}
	return search.matches(row)
}

func (m *multiServer) apiEditor(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	route := strings.TrimPrefix(r.URL.Path, "/api/metadata/editor/")
	switch route {
	case "books":
		m.editorBooks(w, r)
	case "edit":
		m.editorEdit(w, r)
	case "undo":
		m.editorUndoRequest(w, r)
	case "import":
		m.editorImport(w, r)
	case "match":
		m.editorMatch(w, r)
	case "values":
		m.editorValues(w, r)
	case "columns":
		m.editorColumnsRequest(w, r)
	case "count":
		m.editorCount(w, r)
	case "check":
		m.editorCheck(w, r)
	case "tidy":
		m.editorTidy(w, r)
	case "isbn":
		m.editorISBN(w, r)
	case "isbn-ask":
		m.editorISBNAsk(w, r)
	case "match-ask":
		m.editorMatchAsk(w, r)
	case "matches":
		m.editorMatches(w, r)
	case "collections":
		m.editorCollections(w, r)
	case "hardcover-sync":
		m.editorHardcoverSync(w, r)
	case "import-library":
		m.editorImportLibrary(w, r)
	case "import-names":
		m.editorImportNames(w, r)
	case "transfer":
		m.editorTransfer(w, r)
	case "metadata-files":
		m.editorMetadataFiles(w, r)
	case "tasks":
		m.editorTasks(w, r)
	case "match-fields":
		m.editorMatchFields(w, r)
	default:
		// The tasks panel's one write is nested under its tool, so it is matched here instead of as a case:
		// a case per tool would be a second list of tools to keep in step with the first.
		if parts := strings.Split(route, "/"); len(parts) == 3 && parts[0] == "tasks" && parts[2] == "dismiss" {
			m.editorTaskDismiss(w, r, parts[1])
			return
		}
		http.NotFound(w, r)
	}
}

func (m *multiServer) editorLibrary(w http.ResponseWriter, id string) (Library, bool) {
	lib, ok := m.library(id)
	if !ok || lib.Store == nil || !publicationKind(lib.Kind) {
		http.Error(w, "That library has no books to edit.", http.StatusNotFound)
		return Library{}, false
	}
	if m.metadataOverrides == nil {
		http.Error(w, "Edits cannot be saved right now.", http.StatusServiceUnavailable)
		return Library{}, false
	}
	return lib, true
}

func (m *multiServer) editorBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	lib, ok := m.editorLibrary(w, query.Get("library"))
	if !ok {
		return
	}
	// Snapshot, not Books: this request only reads, and Books deep-copies every book (16 000 allocations for
	// 5 400) for a caller that edits nothing. Rows point into the copy the wrapper makes, never into the cache.
	books := publishedBooks(lib.Store)
	issues, facts := editorIssues(books), m.hardcoverFacts()
	custom := map[string]map[string]string{}
	if m.customColumns != nil {
		custom = m.customColumns.Library(lib.ID)
	}
	account := m.readingAccount(r)
	inCollections, within := m.editorMemberships(account, lib), query.Get("collection")
	rows := make([]editorRow, 0, len(books))
	counts := map[string]int{}
	filter, search := query.Get("filter"), newEditorSearch(strings.TrimSpace(query.Get("q")))
	for i := range books {
		book := books[i]
		_, edited := m.metadataOverrides.Get(lib.ID, book.ID)
		row := editorRow{ID: book.ID, Title: book.Title, Authors: append([]string{}, book.Authors...), Series: book.Series, SeriesIndex: book.SeriesIndex, Format: book.Format, Tags: append([]string{}, book.Tags...), Publisher: book.Publisher, Published: book.Published, Language: book.Language,
			Rating: float64(book.Rating) / 2, Description: book.Description, Identifiers: editorIdentifiers(book.Identifiers), TitleSort: book.TitleSort, AuthorSort: strings.Join(book.AuthorSort, " & "),
			Path: book.Path, Edited: edited, Custom: custom[book.ID], book: &books[i]}
		row.Collections = append([]string{}, inCollections.names[book.ID]...)
		if within != "" && !inCollections.books[within][book.ID] {
			continue
		}
		// Everything the background tools know about this book, read once and shared between the badges and the row's
		// own fields. Read after the collection filter so that a book the request drops costs no lookup at all.
		found := m.findings(lib, book, facts)
		row.Issues = found.badges(issues[i], book)
		found.apply(&row)
		if stats, counted := m.editorCounts(lib.ID, book); counted {
			row.Words, row.Pages = stats.Words, stats.Pages
		}
		if m.reading != nil {
			if position, found := m.reading.Get(account, lib.ID, book.ID); found {
				row.Progress = position.Fraction
				if position.Finished {
					row.Progress = 1
				}
			}
		}
		for _, issue := range row.Issues {
			counts[issue]++
		}
		if edited {
			counts["edited"]++
		}
		if editorMatches(row, filter, search) {
			rows = append(rows, row)
		}
	}
	sortEditorRows(rows, query.Get("sort"), query.Get("dir") == "desc")
	offset, _ := strconv.Atoi(query.Get("offset"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	if query.Get("limit") == "all" {
		// One request for the whole matching set, up to editorOneShotMaximum rows.
		//
		// Why it exists: every request builds, filters and sorts *every* row of the library and then slices — so a
		// page load in 1 000-row chunks rebuilt all 5 544 rows of the `test` library six times to send one response's
		// worth of work five times over. Measured on `test` (5 544 books, same bytes either way): one request of all
		// rows 91 ms, versus six chunks of 36 ms — 216 ms of identical rebuilds. Before the row-build work of the same
		// night a chunk cost ~80 ms, so the duplication was costing ~400 ms.
		//
		// The cap is a payload limit, not a performance one: rows measure ~850 bytes each, so this is ~5 MB of JSON in
		// a single response — already what the whole-library view ends up holding in the browser. Past it, the client
		// keeps chunking exactly as before (it compares what it got against the total it was told).
		limit = len(rows)
		if limit > editorOneShotMaximum {
			limit = editorOneShotMaximum
		}
	} else if limit < 1 || limit > editorMaximumPage {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	} else if offset > len(rows) {
		offset = len(rows)
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	// A metadata list is not cacheable: a browser that heuristically caches this GET keeps showing the row as it was
	// before an edit, and the person sees a change they just made disappear. (It also made a deployed fix look like it
	// had not shipped, which is how this was noticed.)
	w.Header().Set("Cache-Control", "no-store")
	// Last, once every field is final including an edited description, and for the rows being sent only: these are
	// notes a person reads, not values the editor sorts or filters on, and this request builds every row of the
	// library to sort it. Building them for the whole library spent most of the work on rows the response drops —
	// measured on 5 430 books, the ISBN note alone was 6 ms and a fifth of the request's allocations to send ~860 of
	// them. The match blurb only rides along for a book that has nothing to describe itself with; deciding that while
	// the row was still being built missed the books whose description comes from an override.
	for position := offset; position < end; position++ {
		row := &rows[position]
		m.addFindingNote(row, lib)
		if row.Match != nil {
			row.Match = row.Match.forList(row)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": len(rows), "library_total": len(books), "library_folder": lib.SourcePath, "collections": inCollections.list, "hardcover_sync": facts.on, "counts": counts, "items": rows[offset:end], "columns": m.editorColumns(), "transfer_to": m.transferTargets(lib)})
}

type editorEditRequest struct {
	Library string `json:"library"`
	// Changes maps book id to field to new value. An empty value resets that field to what the files say.
	Changes map[string]map[string]string `json:"changes"`
	// Reset forgets every edit of these books.
	Reset   []string `json:"reset"`
	Preview bool     `json:"preview"`
}

type editorChange struct {
	Book  string `json:"book"`
	Title string `json:"title"`
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// editorFields are the book fields an edit may change, in the order the editor lists them.
var editorFields = map[string]bool{"title": true, "authors": true, "series": true, "series_index": true, "tags": true, "publisher": true, "published": true,
	"language": true, "rating": true, "description": true, "identifiers": true, "title_sort": true, "author_sort": true}

func editorIdentifiers(identifiers map[string]string) string {
	schemes := make([]string, 0, len(identifiers))
	for scheme := range identifiers {
		schemes = append(schemes, scheme)
	}
	sort.Strings(schemes)
	parts := make([]string, 0, len(schemes))
	for _, scheme := range schemes {
		parts = append(parts, scheme+":"+identifiers[scheme])
	}
	return strings.Join(parts, ", ")
}

func editorValue(book catalog.Book, field string) string {
	switch field {
	case "tags":
		return strings.Join(book.Tags, ", ")
	case "publisher":
		return book.Publisher
	case "published":
		return book.Published
	case "language":
		return book.Language
	case "rating":
		if book.Rating > 0 {
			return strconv.FormatFloat(float64(book.Rating)/2, 'f', -1, 64)
		}
	case "description":
		return book.Description
	case "identifiers":
		return editorIdentifiers(book.Identifiers)
	case "title_sort":
		return book.TitleSort
	case "author_sort":
		return strings.Join(book.AuthorSort, " & ")
	case "title":
		return book.Title
	case "authors":
		return strings.Join(book.Authors, " & ")
	case "series":
		return book.Series
	case "series_index":
		if book.SeriesIndex != nil {
			return strconv.FormatFloat(*book.SeriesIndex, 'f', -1, 64)
		}
	}
	return ""
}

func (m *multiServer) editorEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req editorEditRequest
	if err := decodeJSONRequest(w, r, editorMaximumBody, &req, true); err != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	lib, ok := m.editorLibrary(w, req.Library)
	if !ok {
		return
	}
	if len(req.Changes)+len(req.Reset) == 0 || len(req.Changes)+len(req.Reset) > editorMaximumEdits {
		http.Error(w, fmt.Sprintf("Edit between 1 and %d books at a time.", editorMaximumEdits), http.StatusBadRequest)
		return
	}
	apply := map[string]map[string]string{}
	customApply := map[string]map[string]string{}
	resets := map[string][]string{}
	definitions := map[string]customcolumns.Column{}
	for _, column := range m.editorColumns() {
		definitions[column.ID] = column
	}
	currentCustom := map[string]map[string]string{}
	if m.customColumns != nil {
		currentCustom = m.customColumns.Library(lib.ID)
	}
	var preview []editorChange
	for id, fields := range req.Changes {
		book, found := lib.Store.Book(id)
		if !found {
			http.Error(w, "A book in the request is not in this library.", http.StatusBadRequest)
			return
		}
		for field, value := range fields {
			if strings.HasPrefix(field, "#") {
				column, known := definitions[field]
				normalized, problem := customcolumns.Normalize(column, value)
				if !known || problem != nil {
					message := "That column does not exist."
					if problem != nil {
						message = strings.TrimPrefix(problem.Error(), customcolumns.ErrInvalid.Error()+": ")
					}
					http.Error(w, message, http.StatusBadRequest)
					return
				}
				if customApply[id] == nil {
					customApply[id] = map[string]string{}
				}
				customApply[id][field] = normalized
				preview = append(preview, editorChange{Book: id, Title: book.Title, Field: field, From: currentCustom[id][field], To: normalized})
				continue
			}
			if !editorFields[field] {
				http.Error(w, "That field cannot be edited.", http.StatusBadRequest)
				return
			}
			if calibreOwnsMetadata(lib) && calibreOwnedFields[field] {
				http.Error(w, calibreOwnsMetadataMessage, http.StatusConflict)
				return
			}
			value = strings.TrimSpace(value)
			if field == "series_index" && value != "" {
				if number, err := strconv.ParseFloat(value, 64); err != nil || number <= 0 || math.IsNaN(number) || math.IsInf(number, 0) {
					http.Error(w, "A volume number must be a positive number.", http.StatusBadRequest)
					return
				}
			}
			if problem := editorFieldProblem(field, value); problem != "" {
				http.Error(w, problem, http.StatusBadRequest)
				return
			}
			if field == "title" && value == "" {
				http.Error(w, "A title cannot be empty.", http.StatusBadRequest)
				return
			}
			if value == "" {
				resets[id] = append(resets[id], field)
			} else {
				if apply[id] == nil {
					apply[id] = map[string]string{}
				}
				apply[id][field] = value
			}
			preview = append(preview, editorChange{Book: id, Title: book.Title, Field: field, From: editorValue(book, field), To: value})
		}
	}
	for _, id := range req.Reset {
		if calibreOwnsMetadata(lib) {
			http.Error(w, calibreOwnsMetadataMessage, http.StatusConflict)
			return
		}
		book, found := lib.Store.Book(id)
		if !found {
			http.Error(w, "A book in the request is not in this library.", http.StatusBadRequest)
			return
		}
		preview = append(preview, editorChange{Book: id, Title: book.Title, Field: "all", From: "edited", To: "from the files"})
	}
	sort.SliceStable(preview, func(i, j int) bool { return preview[i].Book < preview[j].Book })
	if req.Preview {
		writeJSON(w, http.StatusOK, map[string]any{"changes": preview})
		return
	}
	m.metadataApplyMu.Lock()
	defer m.metadataApplyMu.Unlock()
	var undos []func() error
	rollback := func(cause error) {
		for i := len(undos) - 1; i >= 0; i-- {
			_ = undos[i]()
		}
		http.Error(w, cause.Error(), http.StatusInternalServerError)
	}
	if len(apply) > 0 {
		undo, err := m.metadataOverrides.ApplyWithUndo(lib.ID, apply)
		if err != nil {
			rollback(err)
			return
		}
		undos = append(undos, undo)
	}
	if len(customApply) > 0 {
		undo, err := m.customColumns.Set(lib.ID, customApply)
		if err != nil {
			rollback(err)
			return
		}
		undos = append(undos, undo)
	}
	for id, fields := range resets {
		undo, err := m.metadataOverrides.Reset(lib.ID, id, fields...)
		if err != nil {
			rollback(err)
			return
		}
		undos = append(undos, undo)
	}
	for _, id := range req.Reset {
		undo, err := m.metadataOverrides.Reset(lib.ID, id)
		if err != nil {
			rollback(err)
			return
		}
		undos = append(undos, undo)
	}
	m.forgetEditorMatches(lib, apply, resets, req.Reset)
	rated := ratedBooks(apply, resets, req.Reset)
	account := m.readingAccount(r)
	token := m.editorUndo.remember(func() error {
		var failure error
		for i := len(undos) - 1; i >= 0; i-- {
			if err := undos[i](); err != nil {
				failure = errors.Join(failure, err)
			}
		}
		if failure == nil {
			m.sendRatingsToHardcover(account, lib, rated)
		}
		return failure
	})
	m.sendRatingsToHardcover(account, lib, rated)
	writeJSON(w, http.StatusOK, map[string]any{"changed": len(preview), "undo": token})
}

func (m *multiServer) editorUndoRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := decodeJSONRequest(w, r, 4096, &req, true); err != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	// The token is only consumed once the undo actually applied, so a transient
	// storage failure leaves the batch retryable instead of answering 410.
	m.metadataApplyMu.Lock()
	defer m.metadataApplyMu.Unlock()
	undo, ok := m.editorUndo.peek(req.Token)
	if !ok {
		http.Error(w, "That change can no longer be undone.", http.StatusGone)
		return
	}
	if err := undo(); err != nil {
		http.Error(w, "The change could not be undone.", http.StatusInternalServerError)
		return
	}
	m.editorUndo.drop(req.Token)
	writeJSON(w, http.StatusOK, map[string]bool{"undone": true})
}
