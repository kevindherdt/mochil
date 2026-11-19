package catalog

import (
	"errors"
	"fmt"
	"strings"
)

func splitOverrideList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// parseOverrideIdentifiers reads "isbn:978..., hardcover:dune" into a map of lowercase scheme to value.
func parseOverrideIdentifiers(raw string) (map[string]string, error) {
	out := map[string]string{}
	for _, part := range splitOverrideList(raw) {
		scheme, id, ok := strings.Cut(part, ":")
		scheme, id = strings.ToLower(strings.TrimSpace(scheme)), strings.TrimSpace(id)
		if !ok || scheme == "" || id == "" || strings.ContainsAny(scheme, " <>&\"'/") {
			return nil, fmt.Errorf("an identifier must look like scheme:value, not %q", part)
		}
		out[scheme] = id
	}
	if len(out) == 0 {
		return nil, errors.New("metadata override identifiers cannot be empty")
	}
	return out, nil
}

func splitOverrideAuthors(value string) []string {
	parts := strings.Split(value, " & ")
	authors := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			authors = append(authors, part)
		}
	}
	return authors
}

// apply returns book with one library's metadata override merged in. It
// never changes the stored override or the caller's book beyond the
// returned value.
func (s *OverrideStore) apply(libraryID string, book Book) Book {
	s.mu.RLock()
	value, ok := s.data.Libraries[libraryID][book.ID]
	s.mu.RUnlock()
	if !ok {
		return book
	}
	if value.Title != "" {
		book.Title = value.Title
	}
	if len(value.Authors) != 0 {
		book.Authors = append([]string(nil), value.Authors...)
		book.AuthorSort = append([]string(nil), value.Authors...)
	}
	if value.Series != "" {
		book.Series = value.Series
	}
	if len(value.Tags) != 0 {
		book.Tags = append([]string(nil), value.Tags...)
	}
	if value.Publisher != "" {
		book.Publisher = value.Publisher
	}
	if value.Published != "" {
		book.Published = value.Published
	}
	if value.Language != "" {
		book.Language = value.Language
	}
	if value.Rating != nil {
		book.Rating = *value.Rating
	}
	if value.Description != "" {
		book.Description = value.Description
	}
	if len(value.Identifiers) != 0 {
		book.Identifiers = map[string]string{}
		for scheme, id := range value.Identifiers {
			book.Identifiers[scheme] = id
		}
	}
	if value.TitleSort != "" {
		book.TitleSort = value.TitleSort
	}
	if value.AuthorSort != "" {
		book.AuthorSort = splitOverrideAuthors(value.AuthorSort)
	}
	if value.SeriesIndex != nil {
		number := *value.SeriesIndex
		book.SeriesIndex = &number
	}
	return book
}
