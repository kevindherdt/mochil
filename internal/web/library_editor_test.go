package web

import (
	"encoding/json"
	"mochil/internal/customcolumns"
	"mochil/internal/enrichment"
	"mochil/internal/ingest"
	"mochil/internal/reading"
	settingssvc "mochil/internal/settings"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/catalog"
)

func editorTestServer(t *testing.T) *multiServer {
	t.Helper()
	server, _ := browseTestServer(t)
	overrides, err := catalog.OpenOverrideStore(filepath.Join(t.TempDir(), "overrides.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.metadataOverrides = overrides
	backend := staticBackend{books: []catalog.Book{
		{ID: "a1", Path: "Dune/Dune 1.epub", Title: "Dune", Authors: []string{"Frank Herbert"}, Series: "Dune", SeriesIndex: ptr(1)},
		{ID: "a2", Path: "Dune/Dune 2.epub", Title: "Dune Messiah", Authors: []string{"Frank Herbert"}, Series: "Dune", SeriesIndex: ptr(1)},
		{ID: "n1", Path: "n.epub", Title: "Neuromancer", Authors: []string{"Unknown"}},
		{ID: "s1", Path: "s.epub", Title: "Roxy (Side Story)", Authors: []string{"A"}, Series: "Mushoku", SeriesIndex: ptr(5)},
		{ID: "u1", Path: "u.epub", Title: "Unnumbered", Authors: []string{"B"}, Series: "Other"},
	}}
	server.libraries = []Library{{ID: "books", Name: "Books", Kind: "books", Store: overrides.Wrap("books", backend)}}
	return server
}

func editorCall(server *multiServer, method, path, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	server.apiEditor(response, httptest.NewRequest(method, path, strings.NewReader(body)))
	return response
}

type editorList struct {
	Total  int            `json:"total"`
	Counts map[string]int `json:"counts"`
	Items  []editorRow    `json:"items"`
}

func listEditor(t *testing.T, server *multiServer, query string) editorList {
	t.Helper()
	response := editorCall(server, http.MethodGet, "/api/metadata/editor/books?library=books"+query, "")
	var out editorList
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &out) != nil {
		t.Fatalf("list = %d %s", response.Code, response.Body.String())
	}
	return out
}

func TestEditorListsBooksWithWhatNeedsAttention(t *testing.T) {
	server := editorTestServer(t)
	all := listEditor(t, server, "")
	if all.Total != 5 || all.Items[0].Series != "Dune" || all.Items[len(all.Items)-1].Series != "" {
		t.Fatalf("order or size wrong: %+v", all.Items)
	}
	if all.Counts["duplicate"] != 2 || all.Counts["no_author"] != 1 || all.Counts["no_number"] != 1 || all.Counts["side_story"] != 1 {
		t.Fatalf("counts = %v", all.Counts)
	}
	if got := listEditor(t, server, "&filter=duplicate"); got.Total != 2 {
		t.Fatalf("duplicates = %d", got.Total)
	}
	if got := listEditor(t, server, "&filter=issues"); got.Total != 4 {
		t.Fatalf("issues = %d, a side story alone is not a problem but the others are", got.Total)
	}
	if got := listEditor(t, server, "&q=herbert+messiah"); got.Total != 1 || got.Items[0].ID != "a2" {
		t.Fatalf("search = %+v", got.Items)
	}
	if got := listEditor(t, server, "&limit=3&offset=3"); len(got.Items) != 2 || got.Total != 5 {
		t.Fatalf("paging = %+v", got)
	}
}

func TestEditorPreviewChangesNothingAndApplyAndResetWork(t *testing.T) {
	server := editorTestServer(t)
	body := `{"library":"books","changes":{"a2":{"series_index":"2"},"n1":{"authors":"William Gibson","series":"Sprawl"}},"preview":true}`
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", body)
	var preview struct{ Changes []editorChange }
	if err := json.Unmarshal(response.Body.Bytes(), &preview); err != nil || len(preview.Changes) != 3 {
		t.Fatalf("preview = %d %s", response.Code, response.Body.String())
	}
	if listEditor(t, server, "&filter=edited").Total != 0 {
		t.Fatal("previewing saved something")
	}
	response = editorCall(server, http.MethodPost, "/api/metadata/editor/edit", strings.Replace(body, `"preview":true`, `"preview":false`, 1))
	if response.Code != http.StatusOK {
		t.Fatalf("apply = %d %s", response.Code, response.Body.String())
	}
	after := listEditor(t, server, "")
	byID := map[string]editorRow{}
	for _, row := range after.Items {
		byID[row.ID] = row
	}
	if *byID["a2"].SeriesIndex != 2 || byID["n1"].Authors[0] != "William Gibson" || byID["n1"].Series != "Sprawl" || !byID["n1"].Edited {
		t.Fatalf("edits not visible: %+v", byID)
	}
	if after.Counts["duplicate"] != 0 || after.Counts["no_author"] != 0 {
		t.Fatalf("fixed problems still counted: %v", after.Counts)
	}
	// an empty value takes just that field back to the files
	editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"n1":{"series":""}}}`)
	if row := listEditor(t, server, "&q=neuromancer").Items[0]; row.Series != "" || row.Authors[0] != "William Gibson" {
		t.Fatalf("partial reset wrong: %+v", row)
	}
	editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","reset":["a2","n1"]}`)
	if listEditor(t, server, "&filter=edited").Total != 0 {
		t.Fatal("reset left edits")
	}
}

func TestEditorUndoTakesBackOnlyTheNewestBatch(t *testing.T) {
	server := editorTestServer(t)
	call := func(body string) string {
		response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", body)
		var out struct{ Undo string }
		if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || out.Undo == "" {
			t.Fatalf("%d %s", response.Code, response.Body.String())
		}
		return out.Undo
	}
	first := call(`{"library":"books","changes":{"a1":{"title":"First edit"}}}`)
	second := call(`{"library":"books","changes":{"a2":{"title":"Second edit"}}}`)
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+first+`"}`); response.Code != http.StatusGone {
		t.Fatalf("an older batch was undone: %d", response.Code)
	}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+second+`"}`); response.Code != http.StatusOK {
		t.Fatalf("undo = %d", response.Code)
	}
	if listEditor(t, server, "&q=first+edit").Total != 1 || listEditor(t, server, "&q=second+edit").Total != 0 {
		t.Fatal("undo took back the wrong batch")
	}
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+first+`"}`); response.Code != http.StatusOK {
		t.Fatalf("the earlier batch is the newest now: %d", response.Code)
	}
}

func TestEditorRejectsBadEdits(t *testing.T) {
	server := editorTestServer(t)
	for name, body := range map[string]string{
		"unknown field":   `{"library":"books","changes":{"a1":{"colour":"red"}}}`,
		"unknown book":    `{"library":"books","changes":{"zzz":{"title":"x"}}}`,
		"bad number":      `{"library":"books","changes":{"a1":{"series_index":"two"}}}`,
		"zero number":     `{"library":"books","changes":{"a1":{"series_index":"0"}}}`,
		"empty title":     `{"library":"books","changes":{"a1":{"title":" "}}}`,
		"nothing":         `{"library":"books"}`,
		"unknown library": `{"library":"nope","changes":{"a1":{"title":"x"}}}`,
		"extra field":     `{"library":"books","changes":{"a1":{"title":"x"}},"surprise":1}`,
	} {
		if response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", body); response.Code < 400 {
			t.Errorf("%s was accepted: %d", name, response.Code)
		}
	}
	if listEditor(t, server, "&filter=edited").Total != 0 {
		t.Fatal("a rejected request saved something")
	}
}

func TestEditorIsOffUntilTheModuleIsOn(t *testing.T) {
	server := editorTestServer(t)
	server.cfg.Modules = map[string]bool{"editor": false}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/metadata/editor/books?library=books", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("module off = %d", response.Code)
	}
}

func TestEditorSortsByAnyColumnBothWays(t *testing.T) {
	server := editorTestServer(t)
	order := func(query string) string {
		var ids []string
		for _, row := range listEditor(t, server, query).Items {
			ids = append(ids, row.ID)
		}
		return strings.Join(ids, ",")
	}
	// books: a1 Dune/1, a2 Dune Messiah/Dune/1, n1 Neuromancer (no author, no series), s1 Roxy/Mushoku/5, u1 Unnumbered/Other
	if got := order(""); got != "a1,a2,s1,u1,n1" {
		t.Fatalf("default (series) order = %s", got)
	}
	if got := order("&sort=title"); got != "a1,a2,n1,s1,u1" {
		t.Fatalf("by title = %s", got)
	}
	if got := order("&sort=title&dir=desc"); got != "u1,s1,n1,a2,a1" {
		t.Fatalf("by title descending = %s", got)
	}
	if got := order("&sort=author"); got != "s1,u1,a1,a2,n1" {
		t.Fatalf("by author = %s (books without an author go last)", got)
	}
	if got := order("&sort=number"); got != "a1,a2,s1,u1,n1" {
		t.Fatalf("by number = %s (books without a number go last)", got)
	}
	if got := order("&sort=nonsense"); got != "a1,a2,s1,u1,n1" {
		t.Fatalf("an unknown column should keep the default order, got %s", got)
	}
}

func TestEditorEditsTheCalibreStyleFields(t *testing.T) {
	server := editorTestServer(t)
	body := `{"library":"books","changes":{"a1":{"tags":"Sci-Fi, Classic","publisher":"Ace","rating":"4.5","identifiers":"isbn:9780441172719","language":"en","description":"Spice.","title_sort":"Dune","author_sort":"Herbert, Frank","published":"1965"}}}`
	if response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", body); response.Code != http.StatusOK {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	var dune editorRow
	for _, row := range listEditor(t, server, "&q=dune").Items {
		if row.ID == "a1" {
			dune = row
		}
	}
	if len(dune.Tags) != 2 || dune.Publisher != "Ace" || dune.Rating != 4.5 || dune.Identifiers != "isbn:9780441172719" || dune.Language != "en" || dune.Description != "Spice." || dune.TitleSort != "Dune" || dune.AuthorSort != "Herbert, Frank" || dune.Published != "1965" {
		t.Fatalf("%+v", dune)
	}
	for name, change := range map[string]string{"stars": `{"rating":"7"}`, "id without scheme": `{"identifiers":"9780441172719"}`, "id with markup": `{"identifiers":"a<b:1"}`} {
		response := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a2":`+change+`}}`)
		if response.Code != http.StatusBadRequest {
			t.Errorf("%s = %d", name, response.Code)
		}
	}
	editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a1":{"rating":"","tags":""}}}`)
	if row := listEditor(t, server, "&q=dune").Items[0]; row.Rating != 0 || len(row.Tags) != 0 || row.Publisher != "Ace" {
		t.Fatalf("empty values should reset only those fields: %+v", row)
	}
	if got := listEditor(t, server, "&sort=publisher&dir=desc").Items[0].ID; got != "a1" {
		t.Fatalf("sorting by publisher, got %s first", got)
	}
}

func TestEditorImportShowsWhatIsNewAndPagesThroughAFilter(t *testing.T) {
	server := editorTestServer(t)
	root := t.TempDir()
	incoming := filepath.Join(root, "incoming")
	for _, name := range []string{"Dune/Dune - Volume 01.epub", "Dune/Dune - Volume 02.epub", "Dune/Dune - Volume 03.epub", "Fresh/Fresh - Volume 01.epub", "Fresh/Fresh - Volume 02.epub"} {
		path := filepath.Join(incoming, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	executor, err := ingest.NewExecutor(incoming, filepath.Join(root, "managed"), ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	server.managedIngest = executor
	type reply struct {
		Enabled bool        `json:"enabled"`
		Items   []importRow `json:"items"`
		Next    string      `json:"next_cursor"`
	}
	get := func(query string) (int, reply) {
		response := editorCall(server, http.MethodGet, "/api/metadata/editor/import?"+query, "")
		var out reply
		_ = json.Unmarshal(response.Body.Bytes(), &out)
		return response.Code, out
	}
	// the test library holds Dune 1 and Dune 1 (as "Dune Messiah"): volume 1 is there, volumes 2 and 3 are new
	code, all := get("library=books&naming=kavita")
	if code != http.StatusOK || len(all.Items) != 5 {
		t.Fatalf("%d %+v", code, all)
	}
	statuses := map[string]string{}
	for _, row := range all.Items {
		statuses[row.SourcePath] = row.Status
	}
	if statuses["Dune/Dune - Volume 01.epub"] != importInLibrary || statuses["Dune/Dune - Volume 02.epub"] != importNewVolume || statuses["Fresh/Fresh - Volume 01.epub"] != importNew {
		t.Fatalf("%v", statuses)
	}
	if all.Items[0].SuggestedTarget == "" || !strings.Contains(all.Items[0].SuggestedTarget, "Vol 0") {
		t.Fatalf("the kavita layout was not used: %q", all.Items[0].SuggestedTarget)
	}
	managedRoot := filepath.Join(root, "managed")
	fresh := editorCall(server, http.MethodGet, "/api/metadata/editor/import?library=books", "")
	if !strings.Contains(fresh.Body.String(), `"managed_directory":`) || strings.Contains(fresh.Body.String(), `"managed_library"`) {
		t.Fatalf("a managed folder that is no library must be reported as such: %s", fresh.Body.String()[:200])
	}
	server.libraries[0].SourcePath = filepath.Dir(managedRoot)
	held := editorCall(server, http.MethodGet, "/api/metadata/editor/import?library=books", "")
	if !strings.Contains(held.Body.String(), `"managed_library":{"id":"books","name":"Books"}`) {
		t.Fatalf("a library holding the managed folder must be named: %s", held.Body.String()[:300])
	}
	server.libraries[0].SourcePath = ""
	_, onlyNew := get("library=books&status=new,new_volume&limit=2")
	if len(onlyNew.Items) != 2 || onlyNew.Next == "" {
		t.Fatalf("filtered page = %+v", onlyNew)
	}
	for _, row := range onlyNew.Items {
		if row.Status == importInLibrary {
			t.Fatalf("an existing book got through the filter: %+v", row)
		}
	}
	_, rest := get("library=books&status=new,new_volume&limit=2&cursor=" + url.QueryEscape(onlyNew.Next))
	seen := map[string]bool{}
	for _, row := range append(onlyNew.Items, rest.Items...) {
		if seen[row.SourcePath] {
			t.Fatalf("%s came back twice across pages", row.SourcePath)
		}
		seen[row.SourcePath] = true
	}
	if len(seen) != 4 {
		t.Fatalf("paging lost books: %v", seen)
	}
	if code, _ = get("library=nope"); code != http.StatusNotFound {
		t.Fatalf("unknown library = %d", code)
	}
	if code, _ = get("naming=../x"); code != http.StatusBadRequest {
		t.Fatalf("bad naming = %d", code)
	}
	server.managedIngest = nil
	if _, off := get("library=books"); off.Enabled || len(off.Items) != 0 {
		t.Fatalf("import should report itself off: %+v", off)
	}
}

func TestEditorImportNamesFollowTheLayoutAndRefuseNamelessBooks(t *testing.T) {
	server := editorTestServer(t)
	body := `{"naming":"kavita","items":[{"source_path":"a/Dune.epub","title":"Dune","authors":["Frank Herbert"],"series":"Dune","series_index":3},{"source_path":"b.epub","title":"  "}]}`
	response := editorCall(server, http.MethodPost, "/api/metadata/editor/import-names", body)
	var out struct {
		Items []struct{ SourcePath, Target string } `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || len(out.Items) != 2 {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if !strings.EqualFold(out.Items[0].Target, "Dune/Dune - Vol 03.epub") || out.Items[1].Target != "" {
		t.Fatalf("%+v", out.Items)
	}
	for name, bad := range map[string]string{"layout": `{"naming":"../x","items":[{"source_path":"a.epub","title":"T"}]}`, "empty": `{"naming":"kavita","items":[]}`, "unknown field": `{"naming":"kavita","items":[{"source_path":"a.epub","title":"T"}],"x":1}`} {
		if r := editorCall(server, http.MethodPost, "/api/metadata/editor/import-names", bad); r.Code != http.StatusBadRequest {
			t.Errorf("%s = %d", name, r.Code)
		}
	}
}

func TestMatchCandidatesAreRankedByTitleThenAuthor(t *testing.T) {
	exact := enrichment.Suggestion{Title: "Dune", Authors: []string{"Frank Herbert"}}
	sequel := enrichment.Suggestion{Title: "Dune Messiah", Authors: []string{"Frank Herbert"}}
	namesake := enrichment.Suggestion{Title: "Dune", Authors: []string{"Someone Else"}}
	other := enrichment.Suggestion{Title: "Neuromancer", Authors: []string{"William Gibson"}}
	authors := []string{"Frank Herbert"}
	scores := []float64{matchScore("Dune", authors, exact), matchScore("Dune", authors, namesake), matchScore("Dune", authors, sequel), matchScore("Dune", authors, other)}
	if scores[0] != 1 || !(scores[0] > scores[1] && scores[1] > scores[3] && scores[2] > scores[3]) {
		t.Fatalf("scores = %v", scores)
	}
	if matchScore("Dune", nil, exact) != 1 {
		t.Fatal("a book with no author should be judged on its title alone")
	}
}

func TestEditorMatchOffersCandidatesAndNeedsHardcover(t *testing.T) {
	server := editorTestServer(t)
	server.hardcover = fakeHardcover(&shelfRecorder{})
	response := editorCall(server, http.MethodGet, "/api/metadata/editor/match?library=books&book=a1", "")
	var out struct {
		Clear      bool `json:"clear"`
		Candidates []struct {
			Title      string  `json:"title"`
			ProviderID string  `json:"provider_id"`
			Score      float64 `json:"score"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || response.Code != http.StatusOK || len(out.Candidates) != 1 {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if out.Candidates[0].Title != "Dune" || out.Candidates[0].ProviderID == "" || out.Candidates[0].Score != 1 || !out.Clear {
		t.Fatalf("%+v", out)
	}
	if r := editorCall(server, http.MethodGet, "/api/metadata/editor/match?library=books&book=nope", ""); r.Code != http.StatusNotFound {
		t.Fatalf("unknown book = %d", r.Code)
	}
	server.cfg.Modules = map[string]bool{"hardcover": false}
	if r := editorCall(server, http.MethodGet, "/api/metadata/editor/match?library=books&book=a1", ""); r.Code != http.StatusConflict {
		t.Fatalf("module off = %d", r.Code)
	}
}

func TestCustomColumnsHoldValuesSortAndUndo(t *testing.T) {
	server := editorTestServer(t)
	store, err := customcolumns.Open(filepath.Join(t.TempDir(), "columns.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.customColumns = store
	created := editorCall(server, http.MethodPost, "/api/metadata/editor/columns", `{"label":"Score","type":"number"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("add = %d %s", created.Code, created.Body.String())
	}
	if bad := editorCall(server, http.MethodPost, "/api/metadata/editor/columns", `{"label":"Score","type":"number"}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("a repeated name = %d", bad.Code)
	}
	if bad := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a1":{"#score":"high"}}}`); bad.Code != http.StatusBadRequest || !strings.Contains(bad.Body.String(), "number") {
		t.Fatalf("text in a number column = %d %s", bad.Code, bad.Body.String())
	}
	if bad := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", `{"library":"books","changes":{"a1":{"#nope":"1"}}}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("an unknown column = %d", bad.Code)
	}
	body := `{"library":"books","changes":{"a1":{"#score":"9"},"a2":{"#score":"10"},"n1":{"#score":"2"}}}`
	if preview := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", strings.Replace(body, "}}}", "}},\"preview\":true}", 1)); preview.Code != http.StatusOK || len(store.Library("books")) != 0 {
		t.Fatalf("a preview must not save: %d %v", preview.Code, store.Library("books"))
	}
	saved := editorCall(server, http.MethodPost, "/api/metadata/editor/edit", body)
	var result struct{ Undo string }
	if err := json.Unmarshal(saved.Body.Bytes(), &result); err != nil || saved.Code != http.StatusOK || result.Undo == "" {
		t.Fatalf("save = %d %s", saved.Code, saved.Body.String())
	}
	asc := listEditor(t, server, "&sort=%23score")
	order := []string{}
	for _, row := range asc.Items[:3] {
		order = append(order, row.ID)
	}
	if strings.Join(order, ",") != "n1,a1,a2" {
		t.Fatalf("a number column must sort as numbers (2, 9, 10): %v", order)
	}
	if last := asc.Items[len(asc.Items)-1]; last.Custom["#score"] != "" {
		t.Fatalf("books without a value come last: %+v", last)
	}
	desc := listEditor(t, server, "&sort=%23score&dir=desc")
	if desc.Items[0].ID != "a2" || desc.Items[len(desc.Items)-1].Custom["#score"] != "" {
		t.Fatalf("descending still keeps empties last: %+v", desc.Items[0])
	}
	if undone := editorCall(server, http.MethodPost, "/api/metadata/editor/undo", `{"token":"`+result.Undo+`"}`); undone.Code != http.StatusOK || len(store.Library("books")) != 0 {
		t.Fatalf("undo = %d %v", undone.Code, store.Library("books"))
	}
	if removed := editorCall(server, http.MethodDelete, "/api/metadata/editor/columns?id=%23score", ""); removed.Code != http.StatusOK || len(store.Columns()) != 0 {
		t.Fatalf("remove = %d", removed.Code)
	}
}

func TestProgressShowsTheReadersPositionAndSorts(t *testing.T) {
	server := editorTestServer(t)
	account := "default"
	if _, err := server.reading.Put(account, reading.Position{LibraryID: "books", BookID: "a1", Fraction: .4}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := server.reading.Put(account, reading.Position{LibraryID: "books", BookID: "a2", Fraction: .9}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := server.reading.SetFinished(account, "books", "n1", true); err != nil {
		t.Fatal(err)
	}
	list := listEditor(t, server, "&sort=progress&dir=desc")
	got := map[string]float64{}
	for _, row := range list.Items {
		got[row.ID] = row.Progress
	}
	if got["a1"] != .4 || got["a2"] != .9 || got["n1"] != 1 || got["u1"] != 0 {
		t.Fatalf("%v", got)
	}
	if list.Items[0].ID != "n1" || list.Items[1].ID != "a2" || list.Items[2].ID != "a1" {
		t.Fatalf("order = %s %s %s", list.Items[0].ID, list.Items[1].ID, list.Items[2].ID)
	}
}

func TestImportDestinationsAreListedAndALibraryCanBeCreated(t *testing.T) {
	server := editorTestServer(t)
	root := t.TempDir()
	incoming, managed := filepath.Join(root, "incoming"), filepath.Join(root, "managed")
	os.MkdirAll(filepath.Join(incoming, "Dune"), 0o700)
	os.WriteFile(filepath.Join(incoming, "Dune", "Dune - Volume 05.epub"), []byte("x"), 0o600)
	executor, err := ingest.NewExecutor(incoming, managed, ingest.NewJournalStore(filepath.Join(root, "journal.json")), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	server.managedIngest = executor
	os.MkdirAll(filepath.Join(managed, "books2"), 0o755)
	os.MkdirAll(filepath.Join(root, "elsewhere"), 0o755)
	server.libraries = append(server.libraries,
		Library{ID: "books2", Name: "Books 2", Kind: "books", SourceType: "filesystem", SourcePath: filepath.Join(managed, "books2")},
		Library{ID: "elsewhere", Name: "Elsewhere", Kind: "books", SourceType: "filesystem", SourcePath: filepath.Join(root, "elsewhere")},
		Library{ID: "cal", Name: "Calibre", Kind: "books", SourceType: "calibre", SourcePath: filepath.Join(managed, "cal")},
		Library{ID: "gone", Name: "Gone", Kind: "books", SourceType: "filesystem", SourcePath: filepath.Join(root, "missing")})
	response := editorCall(server, http.MethodGet, "/api/metadata/editor/import?library=books&naming=kavita", "")
	var out struct {
		Items        []importRow         `json:"items"`
		Destinations []importDestination `json:"destinations"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil || response.Code != http.StatusOK || len(out.Items) != 1 {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if out.Items[0].SuggestedTarget != "Dune/Dune - Vol 05.epub" {
		t.Fatalf("names are the layout's own; the page adds the library folder per book: %q", out.Items[0].SuggestedTarget)
	}
	ids := []string{}
	for _, destination := range out.Destinations {
		ids = append(ids, destination.ID)
	}
	if strings.Join(ids, ",") != "books2,elsewhere" {
		t.Fatalf("every existing folder library of books is a destination, a Calibre library or a missing folder is not: %v", ids)
	}

	// creating a library: folder inside the managed folder, entry added through the settings service
	config := `{"libraries":[{"id":"books","name":"Books","kind":"books","source":{"type":"filesystem","path":"/b"}}],"kosync":{"password":"secret"}}`
	var saved string
	server.settingsService = settingssvc.Service{ReadSource: func() ([]byte, error) { return []byte(config), nil }, SaveSource: func(data []byte) error { saved = string(data); return nil }}
	created := editorCall(server, http.MethodPost, "/api/metadata/editor/import-library", `{"name":"  My New Books! "}`)
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"id":"my-new-books"`) {
		t.Fatalf("%d %s", created.Code, created.Body.String())
	}
	if info, err := os.Stat(filepath.Join(managed, "my-new-books")); err != nil || !info.IsDir() {
		t.Fatalf("the library folder was not made: %v", err)
	}
	if !strings.Contains(saved, `"id": "my-new-books"`) || !strings.Contains(saved, filepath.Join(managed, "my-new-books")) || !strings.Contains(saved, `"id": "books"`) {
		t.Fatalf("the config must keep the old library and gain the new one:\n%s", saved)
	}
	if !strings.Contains(saved, "secret") {
		t.Fatalf("a save through the settings service must keep the saved secrets:\n%s", saved)
	}
	if !strings.Contains(created.Body.String(), `"restart_required":true`) {
		t.Fatalf("a setup without live apply must ask for a restart: %s", created.Body.String())
	}
	live := settingssvc.Service{ReadSource: func() ([]byte, error) { return []byte(config), nil }, ApplySource: func(data []byte) (settingssvc.ApplyResult, error) {
		return settingssvc.ApplyResult{Saved: true, Applied: true}, nil
	}}
	server.settingsService = live
	if appliedNow := editorCall(server, http.MethodPost, "/api/metadata/editor/import-library", `{"name":"Second Shelf"}`); appliedNow.Code != http.StatusCreated || !strings.Contains(appliedNow.Body.String(), `"restart_required":false`) {
		t.Fatalf("a live server must not ask for a restart: %d %s", appliedNow.Code, appliedNow.Body.String())
	}
	if again := editorCall(server, http.MethodPost, "/api/metadata/editor/import-library", `{"name":"Books"}`); again.Code != http.StatusConflict {
		t.Fatalf("a name that is taken = %d", again.Code)
	}
	for name, body := range map[string]string{"empty": `{"name":" "}`, "symbols only": `{"name":"!!!"}`, "extra": `{"name":"x","path":"/etc"}`} {
		if r := editorCall(server, http.MethodPost, "/api/metadata/editor/import-library", body); r.Code != http.StatusBadRequest {
			t.Errorf("%s = %d", name, r.Code)
		}
	}
}
