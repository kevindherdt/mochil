package metadata

import (
	"regexp"
	"strconv"
	"strings"
)

// Tidying: the deterministic half of "clean up what an old import mangled".
//
// The library has titles like "-EIGHTY-SIX - Volume 01" and series that repeat what the title already says. Every one of
// those came from a rule misfiring, and every one can be undone by a rule — which is the point: a rule is reproducible,
// carries its own name so a whole batch can be reverted, runs in microseconds over five thousand rows, and never
// invents a title. A model is left for the rows these rules mark unsure, where judgement is actually needed.
//
// Nothing here writes anything. It returns what it would change and which rule decided it; the editor stages that as a
// normal edit, shows the diff, and applies it through the pipeline that already exists.

// TidyInput is what the rules may look at. Deliberately not a catalog.Book: the rules are pure, so they can be tested
// against rows copied out of the real library without building a catalogue.
type TidyInput struct {
	Title       string
	Series      string
	SeriesIndex *float64
	Authors     []string
	// Role is the catalogue's publication role ("side_story", "volume", …). A rule that changes how a row groups must
	// respect it: a side story is not volume 1 of the main series.
	Role string
}

// TidyResult is the proposed row, plus why. A result that equals the input means "nothing to do".
type TidyResult struct {
	Title       string
	Series      string
	SeriesIndex *float64
	Authors     []string
	// Rules are the ids of the rules that changed something, in the order they ran — so an undo can name them.
	Rules []string
	// Unsure marks a row the rules could not decide confidently. Those are the ones worth asking a model about, and
	// they must never be applied silently.
	Unsure bool
}

// The rule ids. They are user-facing: a preview groups by them ("stripped a leading separator — 4 812") and an undo
// names them ("revert strip-separator").
const (
	RuleStripSeparator = "strip-separator"
	RuleCollapseSpaces = "collapse-spaces"
	RuleDropFormatTag  = "drop-format-tag"
	RuleStripSeries    = "strip-series-prefix"
	RuleSplitVolume    = "split-volume"
	RuleSubSeries      = "sub-series"
	RuleCleanAuthors   = "clean-authors"
	RuleTitleCase      = "title-case" // never on by default: EIGHTY-SIX is stylised, not shouting
)

var (
	// A separator that a bad split left at either end. Includes the en/em dashes a filename may use and the colon a
	// subtitle is often joined with.
	tidyEdgeSeparators = " \t-–—:;,._"

	// "Volume 01", "Vol. 2", "#3", "Book 4" — the forms that say what they mean.
	tidyVolumeMarker = regexp.MustCompile(`(?i)(?:^|[\s\-–—])(?:v(?:ol(?:ume)?)?\.?|book|#)\s*(\d+(?:\.\d+)?)\b`)

	// A trailing distribution tag, not part of any title: "(epub)", "[retail]", "(calibre)", "(v2)".
	tidyFormatTag = regexp.MustCompile(`(?i)\s*[\[(]\s*(?:epub|azw3|mobi|pdf|retail|scan|ocr|calibre|v\d+|fixed|converted)\s*[\])]\s*$`)

	tidySpaces = regexp.MustCompile(`\s{2,}`)
	// A space that should not be there before punctuation: "Vol . 2", "Title : subtitle".
	tidyBeforePunctuation = regexp.MustCompile(`\s+([,.;:!?])`)
	// A volume marker at the very front of what is left after the series name: "Volume 04 - 1932 - Drugs & The Dominos".
	tidyLeadingVolume = regexp.MustCompile(`(?i)^(?:v(?:ol(?:ume)?)?\.?|book)\s*\d+(?:\.\d+)?\b[\s\-–—:;,._]*`)
	// …and whether what is left is nothing but that marker.
	tidyOnlyVolume = regexp.MustCompile(`(?i)^(?:v(?:ol(?:ume)?)?\.?|book|#)?\s*\d+(?:\.\d+)?$`)
	// The number a stylised series name leads with: "86--eighty-six" → "eighty-six".
	seriesLeadingNumber = regexp.MustCompile(`^\d+(?:\s*-+\s*|\s+)`)
)

// Tidy proposes a cleaned row. The second return is false when the row is already clean, so a preview can count what it
// is not going to touch.
func Tidy(in TidyInput) (TidyResult, bool) {
	out := TidyResult{Title: in.Title, Series: in.Series, SeriesIndex: in.SeriesIndex, Authors: append([]string{}, in.Authors...)}
	apply := func(rule string, changed bool) {
		if changed {
			out.Rules = append(out.Rules, rule)
		}
	}

	// 1. The reported defect: separators left at either end by a split. "-EIGHTY-SIX - Volume 01" → "EIGHTY-SIX - Volume
	// 01". A title that is nothing but separators keeps its value rather than becoming empty.
	if trimmed := strings.Trim(out.Title, tidyEdgeSeparators); trimmed != "" && trimmed != out.Title {
		out.Title = trimmed
		apply(RuleStripSeparator, true)
	}
	// Leading only. "-Starting Life in Another World-" is a publisher's stylisation, and trimming its tail renames the
	// series to something the books were not published under.
	if series := strings.TrimLeft(out.Series, tidyEdgeSeparators); series != out.Series {
		out.Series = series
		apply(RuleStripSeparator, true)
	}

	// 2. Doubled spaces and spaces sitting before punctuation — what a filename's underscores turn into.
	if collapsed := tidyBeforePunctuation.ReplaceAllString(tidySpaces.ReplaceAllString(out.Title, " "), "$1"); collapsed != out.Title {
		out.Title = collapsed
		apply(RuleCollapseSpaces, true)
	}

	// 3. A distribution tag on the end is not part of the title.
	if stripped := strings.TrimSpace(tidyFormatTag.ReplaceAllString(out.Title, "")); stripped != "" && stripped != out.Title {
		out.Title = stripped
		apply(RuleDropFormatTag, true)
	}

	// 4–5. "… - Volume 07" is a position, and "… - Side Stories - Volume 02" is a sub-series. Neither touches the title:
	// the volume words stay where the person can read them, because a missing or wrong series position is common and the
	// title is then the only thing that says which book this is.
	// These run before the series prefix is stripped from the title: after that, "EIGHTY-SIX - Side Stories - Volume 02"
	// has already lost the part that says it is a sub-series.
	if out.SeriesIndex == nil || out.Series != "" {
		before, index, _, found := splitVolumeMarker(out.Title)
		if found {
			switch {
			case out.Series == "" && before != "" && !isBareNumber(before):
				// No series yet and the name before the marker reads like one: this row is a volume of it. The title only
				// changes when there is a per-book name after the marker; otherwise it keeps the words the row already has,
				// because "Volume 10" and the bare series name are both worse titles than the one being replaced.
				out.Series = before
				out.SeriesIndex = index
				apply(RuleSplitVolume, true)
			case out.Series != "" && sameName(before, out.Series):
				// The title repeats its own series and then says which volume it is: take the position, and the name after
				// the marker only when one is there.
				out.SeriesIndex = index
				apply(RuleSplitVolume, true)
			case out.Series != "" && hasNamePrefix(before, out.Series):
				// "86--EIGHTY-SIX - Side Stories - Volume 02": the extra part names a sub-series. The title keeps its
				// text — dropping the volume words out of it as well is a second decision, and this rule's job is the
				// grouping.
				out.Series = out.Series + " — " + subSeriesName(before, out.Series)
				out.SeriesIndex = index
				apply(RuleSubSeries, true)
			}
		}
	}

	// 7. Authors: separators left by the same split, blanks, repeats, and "Unknown" next to a real name.
	if cleaned, changed := tidyAuthors(out.Authors); changed {
		out.Authors = cleaned
		apply(RuleCleanAuthors, true)
	}

	// A rule that leaves nothing to call a title has gone too far: keep the original rather than propose an empty cell
	// the editor would reject at apply time anyway.
	if strings.TrimSpace(out.Title) == "" {
		out.Title = in.Title
		out.Unsure = true
	}
	// "Changed" is the difference from the input, not whether a rule ran: a rule that recognised a volume marker already
	// reflected in the row's own position has changed nothing, and counting it would put thousands of no-op rows in front
	// of someone reviewing a diff.
	changed := out.Title != in.Title || out.Series != in.Series || indexText(out.SeriesIndex) != indexText(in.SeriesIndex) || !sameStrings(out.Authors, in.Authors)
	if !changed {
		out.Rules = nil
	}
	return out, changed
}

func indexText(index *float64) string {
	if index == nil {
		return ""
	}
	return strconv.FormatFloat(*index, 'f', -1, 64)
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// splitVolumeMarker finds the last "Volume N"-style marker in a title and returns what is before it, the number, and
// what follows it. The last one is used because a series name may itself contain a marker-shaped word.
func splitVolumeMarker(title string) (before string, index *float64, rest string, found bool) {
	matches := tidyVolumeMarker.FindAllStringSubmatchIndex(title, -1)
	if len(matches) == 0 {
		return "", nil, "", false
	}
	last := matches[len(matches)-1]
	number := title[last[2]:last[3]]
	value, err := strconv.ParseFloat(number, 64)
	if err != nil || value <= 0 {
		return "", nil, "", false
	}
	prefix := strings.Trim(title[:last[0]], tidyEdgeSeparators)
	suffix := strings.Trim(title[last[1]:], tidyEdgeSeparators)
	return prefix, &value, suffix, true
}

// stripLeadingName removes a series name from the front of a title, and reports whether it did.
//
// It matches on words with any separator between them, and against the series' core as well as its full name: the folder
// says "86--EIGHTY-SIX" where the title says "EIGHTY-SIX", and a match that insisted on one spelling would miss the
// rows this exists for. The text it returns keeps the original's case — an earlier version returned the lower-cased
// comparison string, which would have quietly lower-cased a title.
func stripLeadingName(title, name string) (string, bool) {
	for _, candidate := range []string{name, seriesCore(name)} {
		pattern := namePattern(candidate)
		if pattern == nil {
			continue
		}
		match := pattern.FindString(title)
		if match == "" {
			continue
		}
		if rest := strings.Trim(title[len(match):], tidyEdgeSeparators); rest != "" {
			return rest, true
		}
	}
	return title, false
}

// namePattern matches a name at the start of a title, allowing any run of separators between its words and after it.
func namePattern(name string) *regexp.Regexp {
	quoted := make([]string, 0, 4)
	for _, field := range strings.Fields(normaliseName(name)) {
		// A standalone dash is a separator, not a word: as a pattern token it would demand a literal one.
		if strings.Trim(field, "-–—_") == "" {
			continue
		}
		quoted = append(quoted, regexp.QuoteMeta(field))
	}
	if len(quoted) == 0 {
		return nil
	}
	// A dash, colon or semicolon after the name — not a space and not a comma. "Harry Potter and the Order of the
	// Phoenix" and "Totto-chan, the Little Girl at the Window" are one title with the series name inside it, and
	// stripping there invented "and the Order of the Phoenix".
	pattern, err := regexp.Compile(`(?i)^\s*` + strings.Join(quoted, `[\s\-–—_]+`) + `\s*[\-–—:;]\s*`)
	if err != nil {
		return nil
	}
	return pattern
}

// normaliseName lower-cases, folds the dashes a filename may use, and collapses spaces so two spellings of one name
// compare equal without changing the text that is stored.
func normaliseName(value string) string {
	value = strings.NewReplacer("–", "-", "—", "-", "--", "-", "_", " ").Replace(value)
	return tidySpaces.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), " ")
}

func sameName(a, b string) bool { return a != "" && normaliseName(a) == normaliseName(b) }

func hasNamePrefix(value, name string) bool {
	normal := normaliseName(value)
	for _, want := range []string{normaliseName(name), seriesCore(name)} {
		if want != "" && strings.HasPrefix(normal, want+" ") {
			return true
		}
	}
	return false
}

// subSeriesName is the part of `before` that names the sub-series, with the series' own name taken off the front.
func subSeriesName(before, series string) string {
	normal := normaliseName(before)
	for _, want := range []string{normaliseName(series), seriesCore(series)} {
		if want != "" && strings.HasPrefix(normal, want) {
			if rest := strings.Trim(before[len(before)-len(strings.TrimPrefix(normal, want)):], tidyEdgeSeparators); rest != "" {
				return rest
			}
		}
	}
	return strings.Trim(before, tidyEdgeSeparators)
}

// seriesCore drops a leading number from a series name, because a stylised name is often written with it in the folder
// ("86--EIGHTY-SIX") and without it in the title ("EIGHTY-SIX - Volume 01") — the same series, spelled twice.
func seriesCore(name string) string {
	normal := normaliseName(name)
	if core := seriesLeadingNumber.ReplaceAllString(normal, ""); core != normal {
		return strings.Trim(core, tidyEdgeSeparators)
	}
	return ""
}

// isBareNumber reports whether the text before a marker is just a number — "1984 - Volume 1" names no series.
// onlyVolume reports whether what is left after a title's series name is nothing worth keeping as a title: a bare volume
// marker ("Volume 02", "Vol. 2", "Book 3", "07"), or a status word a release group appended
// ("As The Villainess, I Reject These Happy-Bad Endings! - Complete" must not become a book called "Complete").
func onlyVolume(value string) bool {
	trimmed := strings.Trim(strings.TrimSpace(value), tidyEdgeSeparators)
	if tidyOnlyVolume.MatchString(trimmed) {
		return true
	}
	if strings.Trim(trimmed, "+-–—_ .") == "" {
		return true // "+", "++" — a separator between two books, not a name
	}
	words := strings.Fields(strings.ToLower(trimmed))
	if len(words) == 0 || len(words) > 3 {
		return false
	}
	for _, word := range words {
		if !tidyFillerWords[word] {
			return false
		}
	}
	return true
}

// tidyFillerWords are the words a filename appends instead of a title. They are never a book's name, so a rule that
// would leave one of them as the title leaves the title alone instead.
var tidyFillerWords = map[string]bool{
	"complete": true, "completed": true, "omnibus": true, "extra": true, "extras": true,
	"bonus": true, "special": true, "edition": true, "full": true, "unabridged": true,
	"collection": true, "anthology": true, "box": true, "set": true, "retail": true,
}

func isBareNumber(value string) bool {
	_, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// tidyAuthors drops blank and repeated names, strips the separators a split leaves behind, and keeps "Unknown" only
// when it is the only thing there is.
func tidyAuthors(authors []string) ([]string, bool) {
	out := make([]string, 0, len(authors))
	seen := map[string]bool{}
	unknown := -1
	for _, author := range authors {
		// No period in the set: "Soori J." and "Rowling, J. K." are initials, and trimming them renames the author.
		clean := strings.Trim(author, " 	-–—_")
		clean = tidySpaces.ReplaceAllString(clean, " ")
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if key == "unknown" {
			unknown = len(out)
			out = append(out, clean)
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
	}
	if unknown >= 0 && len(out) > 1 {
		out = append(out[:unknown], out[unknown+1:]...)
	}
	if len(out) == 0 {
		return authors, false
	}
	if len(out) == len(authors) {
		for i := range out {
			if out[i] != authors[i] {
				return out, true
			}
		}
		return authors, false
	}
	return out, true
}
