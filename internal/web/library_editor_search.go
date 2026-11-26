package web

import (
	"bytes"
	"strings"
)

// editorSearch is one request's search: the query folded and split once, and each row's text folded into a buffer the
// next row reuses. The shape it replaced built a haystack with strings.ToLower for every book, then lowered and split
// the query again inside the row loop, and allocated a []string for its words — three allocations per book, on a list
// that refetches every chunk of the library for each keystroke the search box debounces.
//
// It folds exactly what the old haystack held: strings.ToLower of the title, the authors joined with spaces, the
// series and the path, in that order, separated by single spaces. Folding each field on its own reads the same as
// folding the joined string, because Unicode lowercasing decides one rune at a time.
type editorSearch struct {
	words [][]byte // folded query words
	text  []byte   // the row's folded fields, reused for the next row
}

// newEditorSearch folds a query into the words every row must contain. It answers nil for a query with no words, and
// nil is what editorMatches treats as "matches everything".
func newEditorSearch(query string) *editorSearch {
	if query == "" {
		return nil
	}
	searcher := &editorSearch{}
	for _, word := range strings.Fields(strings.ToLower(query)) {
		folded := make([]byte, 0, len(word))
		folded, _ = appendFolded(folded, word)
		searcher.words = append(searcher.words, folded)
	}
	if len(searcher.words) == 0 {
		return nil
	}
	return searcher
}

// matches reports whether every word of the query appears in the row.
func (s *editorSearch) matches(row editorRow) bool {
	s.text = s.text[:0]
	s.text, _ = appendFolded(s.text, row.Title)
	s.text = append(s.text, ' ')
	for i, author := range row.Authors {
		if i > 0 {
			s.text = append(s.text, ' ')
		}
		s.text, _ = appendFolded(s.text, author)
	}
	s.text = append(s.text, ' ')
	s.text, _ = appendFolded(s.text, row.Series)
	s.text = append(s.text, ' ')
	s.text, _ = appendFolded(s.text, row.Path)
	for _, word := range s.words {
		if !bytes.Contains(s.text, word) {
			return false
		}
	}
	return true
}
