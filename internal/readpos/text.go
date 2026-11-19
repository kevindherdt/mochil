package readpos

import (
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// chapterText indexes a chapter's readable text twice: "flat" keeps whitespace
// so anchors read as words, and "squashed" is a canonical form with all
// whitespace removed so matching is immune to how different engines separate
// blocks and inline elements.
type chapterText struct {
	body      *node
	flat      []rune
	flatNode  []*node
	flatOff   []int // rune offset inside the owning raw text node
	squashed  []rune
	sqFlat    []int // squashed index -> flat index
	sqString  string
	sqByteAt  []int // squashed rune index -> byte offset in sqString
	textNodes []*node
	firstFlat map[*node]int
}

func newChapterText(body *node) *chapterText {
	c := &chapterText{body: body, firstFlat: map[*node]int{}}
	c.walk(body)
	var builder strings.Builder
	for i, r := range c.flat {
		if unicode.IsSpace(r) {
			continue
		}
		for _, canonicalRune := range canonical(r) {
			if unicode.IsSpace(canonicalRune) || canonicalRune == '­' {
				continue
			}
			c.sqByteAt = append(c.sqByteAt, builder.Len())
			builder.WriteRune(canonicalRune)
			c.squashed = append(c.squashed, canonicalRune)
			c.sqFlat = append(c.sqFlat, i)
		}
	}
	c.sqByteAt = append(c.sqByteAt, builder.Len())
	c.sqString = builder.String()
	return c
}

func (c *chapterText) walk(n *node) {
	if n.isText() {
		c.textNodes = append(c.textNodes, n)
		for offset, r := range []rune(n.text) {
			if r == '­' {
				continue
			}
			if _, seen := c.firstFlat[n]; !seen {
				c.firstFlat[n] = len(c.flat)
			}
			c.flat = append(c.flat, r)
			c.flatNode = append(c.flatNode, n)
			c.flatOff = append(c.flatOff, offset)
		}
		return
	}
	if n.name == "script" || n.name == "style" || n.name == "head" {
		return
	}
	block := blockElements[n.name]
	if block {
		c.separate()
	}
	for _, kid := range n.kids {
		c.walk(kid)
	}
	if block {
		c.separate()
	}
}

// separate keeps words from different blocks apart when the source has no
// whitespace between them. It never maps to a position anyone can land on.
func (c *chapterText) separate() {
	if len(c.flat) == 0 || unicode.IsSpace(c.flat[len(c.flat)-1]) {
		return
	}
	c.flat = append(c.flat, ' ')
	c.flatNode = append(c.flatNode, c.flatNode[len(c.flatNode)-1])
	c.flatOff = append(c.flatOff, c.flatOff[len(c.flatOff)-1])
}

func canonical(r rune) string {
	if r < 0x80 {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		return string(r)
	}
	var builder strings.Builder
	for _, expanded := range strings.ToLower(norm.NFKC.String(string(r))) {
		switch expanded {
		case '‘', '’':
			expanded = '\''
		case '‐', '‑', '‒', '–', '—', '―':
			expanded = '-'
		}
		builder.WriteRune(expanded)
	}
	return builder.String()
}

func squash(text string) string {
	var builder strings.Builder
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		for _, c := range canonical(r) {
			if !unicode.IsSpace(c) && c != '­' {
				builder.WriteRune(c)
			}
		}
	}
	return builder.String()
}

// squashedIndexOfByte converts a byte offset in sqString to a squashed rune index.
func (c *chapterText) squashedIndexOfByte(offset int) int {
	return sort.SearchInts(c.sqByteAt, offset)
}

// occurrences returns the squashed rune index of every match of needle.
func (c *chapterText) occurrences(needle string) []int {
	var found []int
	if needle == "" {
		return nil
	}
	for from := 0; from < len(c.sqString); {
		index := strings.Index(c.sqString[from:], needle)
		if index < 0 {
			break
		}
		found = append(found, c.squashedIndexOfByte(from+index))
		from += index + 1
		for from < len(c.sqString) && c.sqString[from]&0xC0 == 0x80 {
			from++
		}
	}
	return found
}

// squashedAtFlat returns the squashed index of the first non-space character at
// or after a flat position.
func (c *chapterText) squashedAtFlat(flat int) int {
	return sort.SearchInts(c.sqFlat, flat)
}

// words splits flat[from:to] on whitespace.
func (c *chapterText) words(from, to int) []string {
	return strings.Fields(string(c.flat[from:to]))
}
