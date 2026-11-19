package musicnames

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	junkBracket   = regexp.MustCompile(`(?i)\s*[\(\[]\s*(?:official\s+(?:music\s+|lyric\s+)?(?:video|audio)|music\s+video|lyric\s+video|lyrics?|audio|visualizer|hd|hq|4k|explicit|free\s+download|\d{2,3}\s*kbps)\s*[\)\]]`)
	websiteInTag  = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?[a-z0-9][a-z0-9-]*\.(?:com|net|org|ru|info|biz|me|io|fm|to|cc)\b/?`)
	bitrateInTag  = regexp.MustCompile(`(?i)\s*[\(\[]?\s*\d{2,3}\s*kbps\s*[\)\]]?`)
	trackPrefix   = regexp.MustCompile(`^\s*\d{1,3}\s*[-._)]\s*`)
	spacesAndDash = regexp.MustCompile(`\s+`)
)

func hasLower(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func letters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

// sentenceCase turns an ALL CAPS tag into normal capitals. Words of up to three letters stay as they are (OK, DJ, UK),
// as does a single short token such as AC/DC.
func sentenceCase(s string) string {
	if hasLower(s) || letters(s) < 4 || (!strings.ContainsRune(s, ' ') && letters(s) <= 5) {
		return s
	}
	words := strings.Split(s, " ")
	for i, w := range words {
		if letters(w) <= 3 {
			continue
		}
		runes := []rune(strings.ToLower(w))
		for j, r := range runes {
			if unicode.IsLetter(r) && (j == 0 || !unicode.IsLetter(runes[j-1])) {
				runes[j] = unicode.ToUpper(r)
			}
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func tidyText(value string, title bool, artist string) string {
	out := strings.ReplaceAll(value, "_", " ")
	out = websiteInTag.ReplaceAllString(out, " ")
	out = junkBracket.ReplaceAllString(out, "")
	out = bitrateInTag.ReplaceAllString(out, "")
	out = spacesAndDash.ReplaceAllString(strings.TrimSpace(out), " ")
	if title {
		if stripped := strings.TrimSpace(trackPrefix.ReplaceAllString(out, "")); stripped != "" {
			out = stripped
		}
		if artist != "" {
			prefix := strings.ToLower(artist) + " - "
			if lower := strings.ToLower(out); strings.HasPrefix(lower, prefix) && len(out) > len(prefix) {
				out = strings.TrimSpace(out[len(prefix):])
			}
		}
	}
	out = strings.Trim(out, " -_.")
	if out == "" {
		return value
	}
	return sentenceCase(out)
}

// PreClean tidies tags with fixed rules, no model: web addresses, bracketed "official video" and bitrate noise,
// underscores, a leading track number or repeated artist in the title, and ALL CAPS. Words are only ever removed or
// re-capitalised. It returns the cleaned tracks (the input is not changed) and a note per track on what changed.
func PreClean(tracks []Track) ([]Track, [][]string) {
	out := append([]Track(nil), tracks...)
	notes := make([][]string, len(tracks))
	for i := range out {
		artist := firstNonEmpty(out[i].AlbumArtist, out[i].Artist)
		cleanArtist := tidyText(artist, false, "")
		change := func(label string, field *string, title bool) {
			if *field == "" {
				return
			}
			if next := tidyText(*field, title, cleanArtist); next != *field {
				notes[i] = append(notes[i], fmt.Sprintf("%s: %s → %s", label, *field, next))
				*field = next
			}
		}
		change("title", &out[i].Title, true)
		change("album", &out[i].Album, false)
		change("artist", &out[i].Artist, false)
		change("album artist", &out[i].AlbumArtist, false)
	}
	return out, notes
}
