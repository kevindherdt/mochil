package musicnames

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// TidyInstructions asks a language model to clean messy tags before they become file names. It may only remove junk and
// fix capitalisation; Reconcile rejects any cleaned value that contains a word the original did not.
const TidyInstructions = "You clean messy music tags so they can be used in file names. For each numbered track return a cleaned title, artist and album. " +
	"Only REMOVE junk and fix capitalisation: leading track numbers, \"official video\", \"official audio\", \"lyrics\", \"HD\", \"[Remastered]\"-style noise when it is clearly not part of the name, web addresses and site names, " +
	"the artist's name repeated at the start of the title, underscores, and ALL CAPS or all lowercase (use normal title case). " +
	"Never add, translate, guess or reorder words, and never invent a missing value (return an empty string for it). If a value is already clean, return it unchanged. " +
	`Reply with ONLY this JSON and nothing else: {"tracks":[{"n":1,"title":"...","artist":"...","album":"..."}]}`

// TidyMessage lists the tracks by number with their current tags.
func TidyMessage(tracks []Track) string {
	var b strings.Builder
	for i, t := range tracks {
		fmt.Fprintf(&b, "%d. title: %q | artist: %q | album: %q | file: %q\n", i+1, t.Title, firstNonEmpty(t.AlbumArtist, t.Artist), t.Album, t.Path)
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Tidied is the cleaned tags of one track.
type Tidied struct{ Title, Artist, Album string }

// ParseTidy reads the model's JSON, indexed by the track numbers it was given (1-based).
func ParseTidy(reply string) (map[int]Tidied, bool) {
	start, end := strings.Index(reply, "{"), strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return nil, false
	}
	var raw struct {
		Tracks []struct {
			N      int    `json:"n"`
			Title  string `json:"title"`
			Artist string `json:"artist"`
			Album  string `json:"album"`
		} `json:"tracks"`
	}
	if json.Unmarshal([]byte(reply[start:end+1]), &raw) != nil || len(raw.Tracks) == 0 {
		return nil, false
	}
	out := map[int]Tidied{}
	for _, t := range raw.Tracks {
		out[t.N] = Tidied{Title: t.Title, Artist: t.Artist, Album: t.Album}
	}
	return out, true
}

func words(text string) map[string]int {
	counts := map[string]int{}
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		counts[w]++
	}
	return counts
}

// Reconcile returns cleaned if it only removes words from original (or changes their capitalisation or punctuation),
// otherwise original: a model never gets to add or change a word.
func Reconcile(original, cleaned string) string {
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return original
	}
	have := words(original)
	for word, count := range words(cleaned) {
		if have[word] < count {
			return original
		}
	}
	if len(words(cleaned)) == 0 {
		return original
	}
	return cleaned
}

// ApplyTidy returns the tracks with each tag replaced by its reconciled cleaned value, and per track a note on what changed.
func ApplyTidy(tracks []Track, tidy map[int]Tidied) ([]Track, [][]string) {
	out := append([]Track(nil), tracks...)
	notes := make([][]string, len(tracks))
	for i := range out {
		cleaned, ok := tidy[i+1]
		if !ok {
			continue
		}
		change := func(label string, field *string, value string) {
			if next := Reconcile(*field, value); next != *field {
				notes[i] = append(notes[i], fmt.Sprintf("%s: %s → %s", label, *field, next))
				*field = next
			}
		}
		change("title", &out[i].Title, cleaned.Title)
		change("album", &out[i].Album, cleaned.Album)
		if out[i].AlbumArtist != "" {
			change("artist", &out[i].AlbumArtist, cleaned.Artist)
		} else {
			change("artist", &out[i].Artist, cleaned.Artist)
		}
	}
	return out, notes
}
