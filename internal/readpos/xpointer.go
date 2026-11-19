package readpos

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// crengine serialises EPUB positions as
//
//	/body/DocFragment[N]/body/div/p[3]/text()[2].17
//
// Element steps count same-named element siblings and carry "[k]" only when more
// than one exists; text() steps count addressable text children the same way;
// the trailing offset counts characters of the engine's whitespace-collapsed
// text node.
type xpointer struct {
	fragment  int // 1-based DocFragment, i.e. spine position
	steps     []xpStep
	textIndex int // 1-based; 0 when the pointer addresses an element
	offset    int // -1 when absent
}

type xpStep struct {
	name  string
	index int
}

var (
	xpointerPattern = regexp.MustCompile(`^/body(?:\[1\])?/DocFragment(?:\[([0-9]{1,9})\])?(.*)$`)
	elementStep     = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_:-]*)(?:\[([0-9]{1,9})\])?$`)
	textStep        = regexp.MustCompile(`^text\(\)(?:\[([0-9]{1,9})\])?$`)
	offsetSuffix    = regexp.MustCompile(`\.([0-9]{1,9})$`)
)

func parseXPointer(value string) (xpointer, bool) {
	match := xpointerPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return xpointer{}, false
	}
	result := xpointer{fragment: 1, offset: -1}
	if match[1] != "" {
		result.fragment, _ = strconv.Atoi(match[1])
	}
	if result.fragment < 1 {
		return xpointer{}, false
	}
	rest := match[2]
	if suffix := offsetSuffix.FindStringSubmatch(rest); suffix != nil {
		result.offset, _ = strconv.Atoi(suffix[1])
		rest = strings.TrimSuffix(rest, suffix[0])
	}
	for _, segment := range strings.Split(rest, "/") {
		if segment == "" {
			continue
		}
		if text := textStep.FindStringSubmatch(segment); text != nil {
			if result.textIndex != 0 {
				return xpointer{}, false
			}
			result.textIndex = 1
			if text[1] != "" {
				result.textIndex, _ = strconv.Atoi(text[1])
			}
			continue
		}
		step := elementStep.FindStringSubmatch(segment)
		if step == nil || result.textIndex != 0 {
			return xpointer{}, false
		}
		index := 1
		if step[2] != "" {
			index, _ = strconv.Atoi(step[2])
		}
		result.steps = append(result.steps, xpStep{name: strings.ToLower(step[1]), index: index})
	}
	if result.offset >= 0 && result.textIndex == 0 {
		result.textIndex = 1
	}
	return result, true
}

func isXMLSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }

func hasContent(text string) bool {
	return strings.IndexFunc(text, func(r rune) bool { return !unicode.IsSpace(r) }) >= 0
}

// boundary reports whether a block edge touches a text node on one side, where
// the engine discards collapsed whitespace.
func blockEdge(neighbour, parent *node) bool {
	if neighbour == nil {
		return blockElements[parent.name]
	}
	return !neighbour.isText() && blockElements[neighbour.name]
}

func siblingAt(n *node, delta int) *node {
	for i, kid := range n.parent.kids {
		if kid == n {
			if j := i + delta; j >= 0 && j < len(n.parent.kids) {
				return n.parent.kids[j]
			}
			return nil
		}
	}
	return nil
}

// addressable reports whether the engine keeps a text node, and so counts it in
// text() steps. Whitespace between blocks is dropped; whitespace touching an
// inline element is kept.
func addressable(n *node) bool {
	if hasContent(n.text) {
		return true
	}
	for _, delta := range []int{-1, 1} {
		if neighbour := siblingAt(n, delta); neighbour != nil && !neighbour.isText() && !blockElements[neighbour.name] {
			return true
		}
	}
	return false
}

func addressableTexts(element *node) []*node {
	var texts []*node
	for _, kid := range element.kids {
		if kid.isText() && addressable(kid) {
			texts = append(texts, kid)
		}
	}
	return texts
}

// engineOffsets maps each raw rune offset of a text node (0..len) to the
// matching offset in the engine's collapsed text.
func engineOffsets(n *node) []int {
	raw := []rune(n.text)
	offsets := make([]int, len(raw)+1)
	dropLeading := blockEdge(siblingAt(n, -1), n.parent)
	dropTrailing := blockEdge(siblingAt(n, 1), n.parent)
	lastSignificant := len(raw)
	if dropTrailing {
		for lastSignificant > 0 && isXMLSpace(raw[lastSignificant-1]) {
			lastSignificant--
		}
	}
	count, inSpace := 0, false
	for i, r := range raw {
		offsets[i] = count
		switch {
		case isXMLSpace(r):
			if i >= lastSignificant || count == 0 && dropLeading || inSpace {
				continue
			}
			inSpace = true
			count++
		default:
			inSpace = false
			count++
		}
	}
	offsets[len(raw)] = count
	return offsets
}

// flatIndex converts a raw offset inside a text node to a flat index.
func (c *chapterText) flatIndex(n *node, rawOffset int) (int, bool) {
	start, ok := c.firstFlat[n]
	if !ok {
		return 0, false
	}
	removed := 0 // soft hyphens are absent from flat
	for i, r := range []rune(n.text) {
		if i >= rawOffset {
			break
		}
		if r == '­' {
			removed++
		}
	}
	index := start + rawOffset - removed
	return index, true
}

func (c *chapterText) firstTextFlat(element *node) (int, bool) {
	if element.isText() {
		index, ok := c.firstFlat[element]
		return index, ok
	}
	for _, kid := range element.kids {
		if index, ok := c.firstTextFlat(kid); ok {
			return index, true
		}
	}
	return 0, false
}

type quality int

const (
	qualityNone quality = iota
	qualityChapter
	qualityApproximate
	qualityText
)

// resolveXPointer finds the flat text position an xpointer addresses.
func (c *chapterText) resolveXPointer(xp xpointer) (int, quality) {
	current := c.body
	if len(xp.steps) == 0 || xp.steps[0].name != "body" {
		return 0, qualityChapter
	}
	structural := true
	for _, step := range xp.steps[1:] {
		next := nthElement(current, step)
		if next == nil {
			structural = false
			break
		}
		current = next
	}
	if structural && xp.textIndex > 0 {
		texts := addressableTexts(current)
		if xp.textIndex <= len(texts) {
			target := texts[xp.textIndex-1]
			rawOffset := 0
			if xp.offset > 0 {
				offsets := engineOffsets(target)
				rawOffset = sort.SearchInts(offsets, xp.offset)
				if rawOffset >= len(offsets) {
					rawOffset = len(offsets) - 1
				}
			}
			if index, ok := c.flatIndex(target, rawOffset); ok {
				return c.skipSpace(index), qualityText
			}
		}
		structural = false
	}
	if index, ok := c.firstTextFlat(current); ok {
		if structural {
			return c.skipSpace(index), qualityText
		}
		return c.skipSpace(index), qualityApproximate
	}
	return 0, qualityChapter
}

func (c *chapterText) skipSpace(index int) int {
	for index < len(c.flat) && unicode.IsSpace(c.flat[index]) {
		index++
	}
	if index >= len(c.flat) {
		return len(c.flat) - 1
	}
	return index
}

func nthElement(parent *node, step xpStep) *node {
	seen := 0
	for _, kid := range parent.kids {
		if !kid.isText() && kid.name == step.name {
			seen++
			if seen == step.index {
				return kid
			}
		}
	}
	return nil
}

// buildXPointer serialises the engine address of a flat text position.
func (c *chapterText) buildXPointer(fragment, flat int) (string, error) {
	if flat < 0 || flat >= len(c.flat) {
		return "", fmt.Errorf("position %d is outside the chapter", flat)
	}
	target := c.flatNode[flat]
	element := target.parent
	texts := addressableTexts(element)
	textIndex := -1
	for i, text := range texts {
		if text == target {
			textIndex = i
		}
	}
	if textIndex < 0 {
		return "", fmt.Errorf("text node is not addressable")
	}
	var path []string
	for current := element; current != nil && current != c.body.parent; current = current.parent {
		path = append(path, elementSegment(current))
		if current == c.body {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	text := "text()"
	if len(texts) > 1 {
		text = fmt.Sprintf("text()[%d]", textIndex+1)
	}
	offsets := engineOffsets(target)
	rawOffset := min(c.flatOff[flat], len(offsets)-1)
	return fmt.Sprintf("/body/DocFragment[%d]/%s/%s.%d", fragment, strings.Join(path, "/"), text, offsets[rawOffset]), nil
}

func elementSegment(element *node) string {
	if element.parent == nil {
		return element.name
	}
	same, position := 0, 0
	for _, kid := range element.parent.kids {
		if !kid.isText() && kid.name == element.name {
			same++
			if kid == element {
				position = same
			}
		}
	}
	if same > 1 {
		return fmt.Sprintf("%s[%d]", element.name, position)
	}
	return element.name
}
