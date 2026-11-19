package readpos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// TestCorpusRoundTrip is opt-in: READPOS_CORPUS=/path/to/epubs go test -run Corpus.
// For sampled word starts in real books it checks that the position survives
// xpointer -> anchor -> xpointer. Repeated passages may legitimately resolve to
// an earlier twin, so the assertion is on the anchor text, and the hard failure
// rate is reported.
func TestCorpusRoundTrip(t *testing.T) {
	root := os.Getenv("READPOS_CORPUS")
	if root == "" {
		t.Skip("READPOS_CORPUS is not set")
	}
	limit := 40
	var books, samples, unstable, failed int
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".epub") || books >= limit {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()
		info, _ := file.Stat()
		book, err := Open(file, info.Size())
		if err != nil {
			t.Logf("skip %s: %v", filepath.Base(path), err)
			return nil
		}
		books++
		for index := 0; index < len(book.spine); index += max(1, len(book.spine)/6) {
			chapter, err := book.chapter(index)
			if err != nil || len(chapter.flat) < 400 {
				continue
			}
			for _, fraction := range []float64{0.1, 0.5, 0.8} {
				flat := int(float64(len(chapter.flat)) * fraction)
				for flat > 0 && !unicode.IsSpace(chapter.flat[flat-1]) {
					flat--
				}
				flat = chapter.skipSpace(flat)
				original, err := chapter.buildXPointer(index+1, flat)
				if err != nil {
					continue
				}
				samples++
				anchor, level, err := book.AnchorFromXPointer(original)
				if err != nil || level != QualityText {
					failed++
					t.Logf("%s %s: anchor %v level %v", filepath.Base(path), original, err, level)
					continue
				}
				restored, _, err := book.XPointerFromAnchor(anchor)
				if err != nil {
					failed++
					t.Logf("%s %s: restore %v", filepath.Base(path), original, err)
					continue
				}
				if restored != original {
					again, _, _ := book.AnchorFromXPointer(restored)
					if again.Exact != anchor.Exact {
						unstable++
						t.Logf("%s %s -> %s", filepath.Base(path), original, restored)
					}
				}
			}
		}
		return nil
	})
	t.Logf("books=%d samples=%d failed=%d unstable=%d", books, samples, failed, unstable)
	if samples > 0 && float64(failed+unstable)/float64(samples) > 0.02 {
		t.Fatalf("round-trip failure rate too high")
	}
}
