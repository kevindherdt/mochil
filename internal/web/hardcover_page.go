package web

import (
	"net/http"
	"strconv"
	"strings"
)

// hardcoverPage reads the window a list request asks for. A shelf can hold thousands of books, so a list is
// never sent whole: the caller pages through it and filters by title, author or the matched Hardcover title.
type hardcoverPage struct {
	query  string
	limit  int
	offset int
}

func readHardcoverPage(r *http.Request) hardcoverPage {
	page := hardcoverPage{limit: 50}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 && value <= 200 {
			page.limit = value
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			page.offset = value
		}
	}
	page.query = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if len(page.query) > 200 {
		page.query = page.query[:200]
	}
	return page
}

func hardcoverMatchesQuery(page hardcoverPage, fields ...string) bool {
	if page.query == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), page.query) {
			return true
		}
	}
	return false
}

// window returns the slice of items a page asks for, and how many matched in total.
func window[T any](items []T, page hardcoverPage) ([]T, int) {
	total := len(items)
	if page.offset >= total {
		return []T{}, total
	}
	end := page.offset + page.limit
	if end > total {
		end = total
	}
	return items[page.offset:end], total
}

