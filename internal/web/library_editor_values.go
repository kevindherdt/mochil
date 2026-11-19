package web

import (
	"net/http"
	"sort"
	"strings"

	"mochil/internal/catalog"
)

type editorSuggestion struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// editorValueNames lists what a field already holds in the library (an author, a tag, a publisher...), so editing
// can suggest it, and a misspelt twin shows up next to the right one with its own count.
func editorValueNames(books []catalog.Book, field string) (map[string]int, bool) {
	counts := map[string]int{}
	add := func(name string) {
		if name = strings.TrimSpace(name); name != "" {
			counts[name]++
		}
	}
	for _, book := range books {
		switch field {
		case "authors":
			for _, author := range book.Authors {
				add(author)
			}
		case "tags":
			for _, tag := range book.Tags {
				add(tag)
			}
		case "series":
			add(book.Series)
		case "publisher":
			add(book.Publisher)
		case "language":
			add(book.Language)
		default:
			return nil, false
		}
	}
	return counts, true
}

// rankedValues keeps the names that contain the typed letters: those that begin with them first, then the rest,
// each group A to Z.
func rankedValues(counts map[string]int, typed string, limit int) []editorSuggestion {
	typed = strings.ToLower(strings.TrimSpace(typed))
	var begins, contains []editorSuggestion
	for name, count := range counts {
		lower := strings.ToLower(name)
		switch {
		case strings.HasPrefix(lower, typed):
			begins = append(begins, editorSuggestion{name, count})
		case strings.Contains(lower, typed):
			contains = append(contains, editorSuggestion{name, count})
		}
	}
	byName := func(list []editorSuggestion) {
		sort.Slice(list, func(i, j int) bool {
			a, b := strings.ToLower(list[i].Name), strings.ToLower(list[j].Name)
			if a != b {
				return a < b
			}
			return list[i].Name < list[j].Name
		})
	}
	byName(begins)
	byName(contains)
	ranked := append(begins, contains...)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}

func (m *multiServer) editorValues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	lib, ok := m.editorLibrary(w, query.Get("library"))
	if !ok {
		return
	}
	counts, known := editorValueNames(lib.Store.Books(), query.Get("field"))
	if !known {
		http.Error(w, "That field has no suggestions.", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, rankedValues(counts, query.Get("q"), 8))
}
