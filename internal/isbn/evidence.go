package isbn

import (
	"strings"

	"mochil/internal/epubread"
)

// FrontMatterRunes limits the page text kept for a person or helper to read.
const FrontMatterRunes = 2500

// Evidence is what a close call is judged on: for each number found, the text around it, and the text of the copyright
// pages. It is read-only information; nothing in it is saved.
type Evidence struct {
	Snippets    map[string]string // ISBN-13 -> the words around its best sighting
	FrontMatter string
}

// ExtractEvidence is Extract plus the text to judge it by. Only EPUB books carry running text to show.
func ExtractEvidence(file epubread.ReadSeeker, format string, hints Hints) (Result, Evidence, error) {
	if !strings.EqualFold(format, "epub") {
		result, err := Extract(file, format, hints)
		return result, Evidence{}, err
	}
	f, err := searchEPUB(file, hints)
	if err != nil {
		return Result{}, Evidence{}, err
	}
	evidence := Evidence{Snippets: map[string]string{}}
	for isbn, s := range f.snippets {
		evidence.Snippets[isbn] = s.text
	}
	evidence.FrontMatter = strings.TrimSpace(f.front.String())
	if evidence.FrontMatter == "" {
		evidence.FrontMatter = strings.TrimSpace(f.leadingText.String())
	}
	return f.result(), evidence, nil
}

// FindIn returns every correct ISBN (as an ISBN-13) printed in a piece of text.
func FindIn(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, span := range candidatePattern.FindAllStringIndex(text, -1) {
		if isbn, ok := normalizeRun(text[span[0]:span[1]]); ok && !seen[isbn] {
			seen[isbn] = true
			out = append(out, isbn)
		}
	}
	return out
}
