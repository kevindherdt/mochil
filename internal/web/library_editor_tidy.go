package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/metadata"
)

// editorTidy proposes cleaned titles, series and authors for the rows the person selected.
//
// The rules live in internal/metadata/tidying.go and are deterministic; this endpoint is only the door. It answers in the
// same shape the editor already stages (`EditorChanges`: book id → field → value), so the panel's "stage" button hands
// its answer straight to the existing review-and-apply pipeline — nothing here writes anything.
//
// Grouped by rule because that is what makes the preview trustworthy: "stripped a leading separator — 4 812" can be
// accepted or refused as a whole, and the same names are what an undo would quote.
type editorTidyRequest struct {
	Library string   `json:"library"`
	IDs     []string `json:"ids"`
}

type editorTidyGroup struct {
	Rule  string `json:"rule"`
	Count int    `json:"count"`
}

func (m *multiServer) editorTidy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request editorTidyRequest
	if decodeJSONRequest(w, r, editorMaximumBody, &request, false) != nil {
		http.Error(w, "Invalid tidy request", http.StatusBadRequest)
		return
	}
	lib, ok := m.editorLibrary(w, request.Library)
	if !ok {
		return
	}
	if request.Library != "" && calibreSource(lib) {
		// Calibre owns these fields, so proposing them here would be offering work the editor refuses at apply time.
		http.Error(w, "Calibre owns this library's metadata.", http.StatusBadRequest)
		return
	}
	books := publishedBooks(lib.Store)
	wanted := map[string]bool{}
	for _, id := range request.IDs {
		wanted[id] = true
	}
	changes := map[string]map[string]string{}
	groups := map[string]int{}
	unsure := 0
	for i := range books {
		book := &books[i]
		if len(wanted) > 0 && !wanted[book.ID] {
			continue
		}
		result, changed := metadata.Tidy(metadata.TidyInput{
			Title:       book.Title,
			Series:      book.Series,
			SeriesIndex: book.SeriesIndex,
			Authors:     book.Authors,
			Role:        string(book.PublicationRole),
		})
		if result.Unsure {
			unsure++
		}
		if !changed {
			continue
		}
		fields := map[string]string{}
		if result.Title != book.Title {
			fields["title"] = result.Title
		}
		if result.Series != book.Series {
			fields["series"] = result.Series
		}
		if index := tidySeriesIndex(result.SeriesIndex); index != tidySeriesIndex(book.SeriesIndex) {
			fields["series_index"] = index
		}
		if strings.Join(result.Authors, ", ") != strings.Join(book.Authors, ", ") {
			fields["authors"] = strings.Join(result.Authors, ", ")
		}
		if len(fields) == 0 {
			continue
		}
		changes[book.ID] = fields
		for _, rule := range result.Rules {
			groups[rule]++
		}
	}
	order := make([]editorTidyGroup, 0, len(groups))
	keys := make([]string, 0, len(groups))
	for rule := range groups {
		keys = append(keys, rule)
	}
	sort.Strings(keys)
	for _, rule := range keys {
		order = append(order, editorTidyGroup{Rule: rule, Count: groups[rule]})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"changes": changes,
		"groups":  order,
		"books":   len(changes),
		"unsure":  unsure,
	})
}

// tidySeriesIndex is how a position is spelled in the editor's own cells: a bare number, or empty for none.
func tidySeriesIndex(index *float64) string {
	if index == nil {
		return ""
	}
	return strconv.FormatFloat(*index, 'f', -1, 64)
}

// calibreSource reports whether this library is Calibre's own, whose metadata the editor treats as read-only.
func calibreSource(lib Library) bool {
	return lib.SourceType == "calibre"
}
