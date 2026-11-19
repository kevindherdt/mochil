package metadata

import (
	"errors"
	"sort"

	"mochil/internal/catalog"
)

// BookChanges is the storage-independent result of a resolved review. Storage
// adapters consume this plan without needing to understand issue codes or job
// decision semantics.
type BookChanges struct {
	Book   catalog.Book
	Fields map[string]string
}

// FieldForIssue maps review vocabulary to normalized metadata fields.
func FieldForIssue(code string) string {
	switch code {
	case "missing_title":
		return "title"
	case "missing_author":
		return "authors"
	case "missing_series_index", "duplicate_series_index":
		return "series_index"
	default:
		return ""
	}
}

// BuildChangeSet validates all accepted decisions and groups them by book. It
// is intentionally unaware of filesystem, Calibre, and catalogue-override
// mechanics; those adapters receive the exact same deterministic plan.
func BuildChangeSet(job Job, backend catalog.Backend) ([]BookChanges, error) {
	if backend == nil {
		return nil, errors.New("the reviewed library is unavailable")
	}
	grouped := make(map[string]*BookChanges)
	for _, item := range job.Items {
		if item.Decision != "accepted" {
			continue
		}
		book, found := backend.Book(item.BookID)
		field := FieldForIssue(item.Code)
		if !found || field == "" {
			return nil, errors.New("the apply plan contains an unavailable or unsupported publication")
		}
		target := grouped[book.ID]
		if target == nil {
			target = &BookChanges{Book: book, Fields: make(map[string]string)}
			grouped[book.ID] = target
		}
		if prior, duplicate := target.Fields[field]; duplicate && prior != item.DecisionValue {
			return nil, errors.New("the apply plan contains conflicting values for one field")
		}
		target.Fields[field] = item.DecisionValue
	}
	if len(grouped) == 0 {
		return nil, errors.New("the review has no accepted changes")
	}
	result := make([]BookChanges, 0, len(grouped))
	for _, target := range grouped {
		result = append(result, *target)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Book.ID < result[j].Book.ID })
	return result, nil
}
