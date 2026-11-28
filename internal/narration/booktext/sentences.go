package booktext

import (
	"regexp"
	"strings"
	"unicode"
)

// space is what JavaScript's \s matches; the regexp package's \s is ASCII only, which would split
// sentences differently from the web reader around no-break spaces.
const space = `[\s\x{0b}\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}]`

var (
	abbreviation = regexp.MustCompile(`(?i)\b(?:Mr|Mrs|Ms|Dr|Prof|St|Sr|Jr|vs|etc|e\.g|i\.e|dhr|mevr)$`)
	initials     = regexp.MustCompile(`(?:\b[A-Za-z]\.)+[A-Za-z]$`)
	// The terminators include the fullwidth CJK forms, which do not need trailing
	// whitespace: a Chinese or Japanese paragraph would otherwise be hard-cut at
	// maxChars instead of split at its own full stops.
	sentenceEnd = regexp.MustCompile(`[.!?…]["'”’»)\]）]*(?:` + space + `+|$)|[。！？]["'”’»)\]）]*`)
	runsOfSpace = regexp.MustCompile(space + `+`)
)

// hasWord reports whether text carries something word-like. Go's \w is ASCII
// only, so a regexp here dropped every Chinese, Japanese, Korean, Thai or
// Cyrillic sentence and left read-along chapters with no text at all.
func hasWord(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// clippedFragment reports a piece too short to hear on its own. Space-separated
// languages are judged by words; scripts written without spaces are judged by
// runes, because every CJK sentence counts as one "word" and would otherwise be
// merged into the rest of the chapter.
func clippedFragment(text string) bool {
	trimmed := strings.TrimSpace(text)
	if strings.IndexFunc(trimmed, unicode.IsSpace) >= 0 || strings.IndexFunc(trimmed, func(r rune) bool { return unicode.Is(unicode.Latin, r) }) >= 0 {
		return len(strings.Fields(trimmed)) < minWords
	}
	if strings.IndexFunc(trimmed, unicode.IsLetter) < 0 {
		return false
	}
	return len([]rune(trimmed)) < 6
}

const (
	minWords = 3
	maxChars = 400
)

// trailingToken returns at most width bytes of text ending at end, widened leftwards to a word boundary so a window never
// begins inside the token it is testing. The abbreviation and initials patterns are anchored at the end ($), which is what
// makes a bounded window equivalent to the whole prefix.
func trailingToken(text string, end, width int) string {
	start := end - width
	if start < 0 {
		start = 0
	}
	for start > 0 && end-start < width*4 && isWordByte(text[start-1]) {
		start--
	}
	return text[start:end]
}

func isWordByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '.'
}

// span is a [from, to) range of runes in a block's text.
type span struct{ from, to int }

// spans cuts a block's text into one range per sentence, as sentenceSpans does in the web reader.
func spans(runes []rune) []span {
	text := string(runes)
	// Regular expressions report byte offsets; sentences are cut by rune.
	byteToRune := make(map[int]int, len(runes)+1)
	at := 0
	for i, r := range runes {
		byteToRune[at] = i
		at += len(string(r))
	}
	byteToRune[at] = len(runes)
	_ = at

	var cut []span
	start := 0
	for _, m := range sentenceEnd.FindAllStringIndex(text, -1) {
		if text[m[0]] == '.' {
			// A bounded window, not the whole prefix: both patterns are anchored at the end, so the answer is the same,
			// but re-scanning the prefix for every full stop made a single-block chapter quadratic -- 400KB took 92s.
			before := trailingToken(text, m[0], 32)
			if abbreviation.MatchString(before) || initials.MatchString(before) {
				continue
			}
		}
		end := byteToRune[m[1]]
		// “Where?” she asked: a lowercase word after the closing quote continues the sentence.
		if end < len(runes) && unicode.IsLower(runes[end]) {
			continue
		}
		cut = append(cut, span{start, end})
		start = end
	}
	if strings.TrimSpace(string(runes[start:])) != "" {
		cut = append(cut, span{start, len(runes)})
	}
	// A greeting or an exclamation alone sounds clipped; read it with what follows.
	var merged []span
	for _, s := range cut {
		if n := len(merged); n > 0 && clippedFragment(string(runes[merged[n-1].from:merged[n-1].to])) {
			merged[n-1].to = s.to
		} else {
			merged = append(merged, s)
		}
	}
	// A very long sentence is read in parts, at commas, so audio starts sooner and stays within limits.
	var parts []span
	for _, s := range merged {
		cursor := s.from
		for s.to-cursor > maxChars {
			window := string(runes[cursor : cursor+maxChars])
			cutAt := max(lastIndexRunes(window, ", "), lastIndexRunes(window, "; "), lastIndexRunes(window, ": "))
			var end int
			if cutAt > maxChars/3 {
				end = cursor + cutAt + 2
			} else if space := lastIndexRunes(window, " "); space > 0 {
				end = cursor + space + 1
			} else {
				end = cursor + maxChars
			}
			parts = append(parts, span{cursor, end})
			cursor = end
		}
		parts = append(parts, span{cursor, s.to})
	}
	var out []span
	for _, p := range parts {
		if hasWord(string(runes[p.from:p.to])) {
			out = append(out, p)
		}
	}
	return out
}

// lastIndexRunes is strings.LastIndex, counted in runes (-1 when absent).
func lastIndexRunes(text, find string) int {
	i := strings.LastIndex(text, find)
	if i < 0 {
		return -1
	}
	return len([]rune(text[:i]))
}

// Lead limits for cutting a long sentence after its opening clause.
const (
	leadMin = 25
	leadMax = 90
	restMin = 30
)

var leadMark = regexp.MustCompile(`[,;:]` + space)

// leadEnd is where a long sentence's opening clause ends (in runes), or 0 for a sentence that is not cut.
func leadEnd(raw []rune) int {
	if len(raw) <= leadMin {
		return 0
	}
	// Search from leadMin on, as the web reader does.
	tail := string(raw[leadMin:])
	m := leadMark.FindStringIndex(tail)
	if m == nil {
		return 0
	}
	index := leadMin + len([]rune(tail[:m[0]]))
	if index < leadMax && len(raw)-index > restMin {
		return index + 1
	}
	return 0
}
