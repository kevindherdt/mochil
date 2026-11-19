package document

import (
	"path/filepath"
	"testing"
)

// Word repeats a text box or drawing once per rendering choice: the modern copy
// inside mc:Choice and a legacy copy inside mc:Fallback. Both carry the same
// text, so reading the fallback doubles every word in those paragraphs.
func TestReadDOCXCountsAlternateContentOnce(t *testing.T) {
	root := t.TempDir()
	writeTestDOCX(t, filepath.Join(root, "Boxed.docx"), `<w:document xmlns:w="urn:word" xmlns:mc="urn:mc" xmlns:wps="urn:wps">
<w:body>
<w:p><w:r><w:t>Before</w:t></w:r></w:p>
<w:p><w:r><mc:AlternateContent>
<mc:Choice Requires="wps"><w:drawing><wps:txbx><w:txbxContent><w:p><w:r><w:t>Boxed text</w:t></w:r></w:p></w:txbxContent></wps:txbx></w:drawing></mc:Choice>
<mc:Fallback><w:pict><w:txbxContent><w:p><w:r><w:t>Boxed text</w:t></w:r></w:p></w:txbxContent></w:pict></mc:Fallback>
</mc:AlternateContent></w:r></w:p>
<w:p><w:r><w:t>After</w:t></w:r></w:p>
</w:body></w:document>`)
	store := NewStore("notes", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	content, _, err := store.ReadText(store.Items()[0].ID, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if got := countOccurrences(string(content), "Boxed text"); got != 1 {
		t.Fatalf("text = %q, want the boxed text once, found %d times", content, got)
	}
}

func countOccurrences(haystack, needle string) int {
	count := 0
	for offset := 0; offset < len(haystack); {
		index := indexOf(haystack[offset:], needle)
		if index < 0 {
			break
		}
		count++
		offset += index + len(needle)
	}
	return count
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
