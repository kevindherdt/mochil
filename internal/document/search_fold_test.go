package document

import (
	"strings"
	"testing"
)

// Searching used to lower-case every indexed document on every query, under the read lock. The folded copy is built once.
func TestSearchUsesTheFoldedCopyAndFindsMixedCase(t *testing.T) {
	index := &TextIndex{entries: map[string]textEntry{
		"note": {ID: "note", Content: "The Lighthouse Keeper was here", Folded: "the lighthouse keeper was here"},
		"big":  {ID: "big", Content: strings.Repeat("filler text ", 500) + "HIDDEN NEEDLE " + strings.Repeat("more filler ", 500), Folded: strings.ToLower(strings.Repeat("filler text ", 500) + "HIDDEN NEEDLE " + strings.Repeat("more filler ", 500))},
	}}
	matches := index.searchLocked("lighthouse")
	if _, ok := matches["note"]; !ok {
		t.Fatalf("mixed-case document was not found: %v", matches)
	}
	// The snippet around a match deep inside a long document must still be the words around it.
	snippet, ok := index.searchLocked("hidden needle")["big"]
	if !ok || !strings.Contains(strings.ToLower(snippet), "hidden needle") {
		t.Fatalf("snippet = %q (found=%v)", snippet, ok)
	}
	if len([]rune(snippet)) > 200 {
		t.Fatalf("snippet is %d runes; it should be a window, not the document", len([]rune(snippet)))
	}
}

// An entry loaded from a cache written before the field existed is folded on demand rather than searched for in its raw form.
func TestSearchFoldsAnEntryWithoutTheCachedCopy(t *testing.T) {
	index := &TextIndex{entries: map[string]textEntry{"note": {ID: "note", Content: "Mixed CASE Content"}}}
	if _, ok := index.searchLocked("mixed case")["note"]; !ok {
		t.Fatalf("entry without a cached folded copy was not found")
	}
}

func BenchmarkSearchTwoThousandNotes(b *testing.B) {
	body := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 90) // about 4 KB
	index := &TextIndex{entries: make(map[string]textEntry, 2000)}
	for i := 0; i < 2000; i++ {
		id := string(rune('a'+i%26)) + strings.Repeat("x", 6) + string(rune('0'+i%10))
		index.entries[id] = textEntry{ID: id, Content: body, Folded: strings.ToLower(body)}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index.searchLocked("brown fox")
	}
}
