package readpos

import (
	"math"
	"strings"
	"unicode"
)

// Anchor mirrors the web reader's text anchor: the words that start at a
// location, the words before and after for disambiguation, and the spine item
// they live in.
type Anchor struct {
	SpineIndex int
	Prefix     string
	Exact      string
	Suffix     string
	Progress   float64
}

const (
	prefixWords      = 24
	exactWords       = 16
	suffixWords      = 32
	minimumExactRune = 20 // squashed; shorter quotes are not unique enough
	searchWindow     = 4000
)

// anchorAt reads an anchor at a flat position, snapped to the start of its word.
func (c *chapterText) anchorAt(flat int) (prefix, exact, suffix string, ok bool) {
	if flat < 0 || flat >= len(c.flat) {
		return "", "", "", false
	}
	for flat > 0 && !unicode.IsSpace(c.flat[flat-1]) {
		flat--
	}
	from := max(0, flat-searchWindow)
	to := min(len(c.flat), flat+searchWindow)
	before := c.words(from, flat)
	if from > 0 && len(before) > 0 {
		before = before[1:] // the first word of a cut window may be partial
	}
	after := c.words(flat, to)
	if to < len(c.flat) && len(after) > 0 {
		after = after[:len(after)-1]
	}
	if len(before) > prefixWords {
		before = before[len(before)-prefixWords:]
	}
	exactEnd := min(len(after), exactWords)
	suffixEnd := min(len(after), exactWords+suffixWords)
	exact = strings.Join(after[:exactEnd], " ")
	if len([]rune(squash(exact))) < minimumExactRune {
		return "", "", "", false
	}
	return strings.Join(before, " "), exact, strings.Join(after[exactEnd:suffixEnd], " "), true
}

type match struct {
	flat  int
	score float64
	sq    int
}

// findAnchor scores every occurrence of the quoted text. approximate is the
// anchor's book fraction and chapterFraction maps a squashed index to one, so
// repeated sentences resolve to the occurrence nearest where the reader was.
func (c *chapterText) findAnchor(a Anchor, preferred bool, fraction func(sq int) float64) (match, bool) {
	exact := squash(a.Exact)
	if len([]rune(exact)) < minimumExactRune {
		return match{}, false
	}
	prefix, suffix := squash(a.Prefix), squash(a.Suffix)
	best, found := match{}, false
	for _, index := range c.occurrences(exact) {
		score := 100.0
		if preferred {
			score += 50
		}
		score -= math.Abs(fraction(index)-a.Progress) * 20
		before := string(c.squashed[max(0, index-len([]rune(prefix))):index])
		afterEnd := min(len(c.squashed), index+len([]rune(exact))+len([]rune(suffix)))
		after := string(c.squashed[min(len(c.squashed), index+len([]rune(exact))):afterEnd])
		if prefix != "" && strings.HasSuffix(before, prefix) {
			score += 35
		}
		if suffix != "" && strings.HasPrefix(after, suffix) {
			score += 35
		}
		if !found || score > best.score {
			best, found = match{flat: c.sqFlat[index], score: score, sq: index}, true
		}
	}
	return best, found
}
