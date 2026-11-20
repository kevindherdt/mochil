package web

// editorMatchInputs are the fields a Hardcover lookup is built from. Changing one
// makes whatever the last lookup learned about this file stale, so the cache entry
// is forgotten and the next match run looks the book up again.
// calibreOwnsMetadataMessage explains why the editor will not write Calibre's fields.
const calibreOwnsMetadataMessage = "This library's metadata is managed by Calibre; change it there, or use a library that Mochil owns."

// calibreOwnsMetadata reports whether Calibre owns this library's metadata. An override
// written here would disagree with Calibre's own database and be overwritten by it.
func calibreOwnsMetadata(lib Library) bool { return lib.SourceType == "calibre" }

var editorMatchInputs = map[string]bool{"title": true, "authors": true, "series": true, "isbn": true, "identifiers": true}

// calibreOwnedFields is what the editor must refuse on a Calibre library. It is NOT editorMatchInputs: that map describes
// what a Hardcover lookup is built from, and reusing it let the series keep its guard while the volume number beside it
// (series_index) and the sort keys walked through — overrides Calibre's own database would silently overwrite. Rating,
// tags, publisher and the rest stay editable by design.
var calibreOwnedFields = map[string]bool{
	"title": true, "authors": true, "series": true, "series_index": true,
	"isbn": true, "identifiers": true, "title_sort": true, "author_sort": true,
}

func touchesMatchInputs(fields []string) bool {
	for _, field := range fields {
		if editorMatchInputs[field] {
			return true
		}
	}
	return false
}

// forgetEditorMatches drops cached lookup results for the books an edit changed in
// a way the lookup depends on. The cache is keyed on file size and modification
// time, which a metadata edit does not change.
func (m *multiServer) forgetEditorMatches(lib Library, apply map[string]map[string]string, resets map[string][]string, resetAll []string) {
	if m.matchFound == nil {
		return
	}
	changed := false
	for id, fields := range apply {
		names := make([]string, 0, len(fields))
		for field := range fields {
			names = append(names, field)
		}
		if touchesMatchInputs(names) && m.matchFound.Remove(lib.ID, id) {
			changed = true
		}
	}
	for id, fields := range resets {
		if touchesMatchInputs(fields) && m.matchFound.Remove(lib.ID, id) {
			changed = true
		}
	}
	for _, id := range resetAll {
		if m.matchFound.Remove(lib.ID, id) {
			changed = true
		}
	}
	if changed {
		_ = m.matchFound.Save()
	}
}
