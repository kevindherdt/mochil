package video

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	matchBracketTags = regexp.MustCompile(`\s*\[[^\]]*]`)
	matchReleaseTags = regexp.MustCompile(`(?i)[\s._-]*\b(?:blu-?ray|bdrip|brrip|web[-\s.]?dl|webrip|hdtv|dvdrip|remux)\b[-\s.]*(?:\d{3,4}p)?`)
	matchSpecTags    = regexp.MustCompile(`(?i)[\s._-]*\b(?:2160p|1080p|720p|480p|4k|hdr10?|x26[45]|hevc)\b`)
	matchYear        = regexp.MustCompile(`\s*[(\[]((?:19|20)\d{2})[)\]]`)
)

// ParseMediaName splits a library name such as "The Witch - Part 1 The Subversion (2018) Bluray-1080p" into the title a
// metadata search should use and the year it carries, if any.
func ParseMediaName(raw string) (title string, year int) {
	value := matchBracketTags.ReplaceAllString(raw, "")
	value = matchReleaseTags.ReplaceAllString(value, "")
	value = matchSpecTags.ReplaceAllString(value, "")
	if found := matchYear.FindStringSubmatch(value); found != nil {
		year, _ = strconv.Atoi(found[1])
		value = matchYear.ReplaceAllString(value, "")
	}
	value = strings.Trim(strings.TrimSpace(value), "-._ ")
	if value == "" {
		return strings.TrimSpace(raw), 0
	}
	return value, year
}

// NormalizeTitle reduces a title to lower-case letters and digits so "Spider-Man: Homecoming" and
// "spider man homecoming" compare equal.
func NormalizeTitle(value string) string {
	value = strings.ReplaceAll(strings.ToLower(value), "&", " and ")
	var out strings.Builder
	space := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			out.WriteRune(r)
			space = false
		case !space && out.Len() > 0:
			out.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(out.String())
}

// ConfidentMatch says whether a search result is the title the library name asks for: the same words, and the same
// year when the name states one. Anything less is left for a person to decide.
func ConfidentMatch(wantTitle string, wantYear int, gotTitle string, gotYear int) bool {
	want, got := NormalizeTitle(wantTitle), NormalizeTitle(gotTitle)
	if want == "" {
		return false
	}
	if want == got {
		return wantYear == 0 || gotYear == 0 || wantYear == gotYear
	}
	// "When They Cry - Higurashi" and "Higurashi: When They Cry" are one show; reordered words count only when both years are stated and agree.
	return wantYear != 0 && wantYear == gotYear && sortedWords(want) == sortedWords(got)
}

func sortedWords(normalized string) string {
	words := strings.Fields(normalized)
	sort.Strings(words)
	return strings.Join(words, " ")
}
