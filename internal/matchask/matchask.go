// Package matchask lets a language model settle a close call between Hardcover candidates for one book. The automatic
// title search already ranks the candidates; the model only chooses among them, and Verify rejects anything else.
package matchask

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Instructions is the system message. The model may only name a candidate by its number.
const Instructions = "You help match an ebook in someone's library to the right entry in the Hardcover book database. " +
	"Choose the candidate that is the SAME work as the library book: same title (allowing subtitles, series prefixes and edition wording), same author. " +
	"A different book in the same series, a study guide, summary, box set, omnibus or a book by another author is NOT a match. " +
	"For a series volume the candidate must be that volume. You may answer only with the number of a listed candidate. " +
	"If none is clearly the same work, or you are unsure, answer null. Reply with ONLY this JSON and nothing else: " +
	`{"pick":<candidate number> or null,"confidence":"high" or "medium" or "low","reason":"<one short sentence>"}`

// Book is what the library knows about the book.
type Book struct {
	Title       string
	Authors     string
	Series      string
	Volume      *float64
	Published   string
	Description string
}

// Candidate is one Hardcover result.
type Candidate struct {
	Title    string
	Authors  string
	Year     int
	Overview string
	Score    float64
}

func clip(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(text); len(r) > limit {
		return string(r[:limit]) + "…"
	}
	return text
}

// Message lists the book and its numbered candidates.
func Message(book Book, candidates []Candidate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Library book: %s\n", book.Title)
	if book.Authors != "" {
		fmt.Fprintf(&b, "Author: %s\n", book.Authors)
	}
	if book.Series != "" {
		fmt.Fprintf(&b, "Series: %s", book.Series)
		if book.Volume != nil {
			fmt.Fprintf(&b, ", volume %s", strconv.FormatFloat(*book.Volume, 'f', -1, 64))
		}
		b.WriteString("\n")
	}
	if book.Published != "" {
		fmt.Fprintf(&b, "Published: %s\n", book.Published)
	}
	if book.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", clip(book.Description, 300))
	}
	b.WriteString("\nHardcover candidates (best title match first):\n")
	for index, c := range candidates {
		fmt.Fprintf(&b, "%d. %s", index+1, c.Title)
		if c.Authors != "" {
			fmt.Fprintf(&b, " — by %s", c.Authors)
		}
		if c.Year > 0 {
			fmt.Fprintf(&b, " (%d)", c.Year)
		}
		fmt.Fprintf(&b, " [title match %d%%]\n", int(c.Score*100))
		if c.Overview != "" {
			fmt.Fprintf(&b, "   %s\n", clip(c.Overview, 200))
		}
	}
	return b.String()
}

// Answer is what the model replied. Pick is 1-based; 0 means it declined.
type Answer struct {
	Pick       int
	Confidence string
	Reason     string
}

// ParseAnswer reads the JSON object out of a reply that may have words or a code fence around it.
func ParseAnswer(reply string) (Answer, bool) {
	start, end := strings.Index(reply, "{"), strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return Answer{}, false
	}
	var raw struct {
		Pick       any    `json:"pick"`
		Confidence string `json:"confidence"`
		Reason     string `json:"reason"`
	}
	if json.Unmarshal([]byte(reply[start:end+1]), &raw) != nil {
		return Answer{}, false
	}
	answer := Answer{Confidence: strings.ToLower(strings.TrimSpace(raw.Confidence)), Reason: clip(raw.Reason, 240)}
	switch value := raw.Pick.(type) {
	case float64:
		answer.Pick = int(value)
	case string:
		answer.Pick, _ = strconv.Atoi(strings.TrimSpace(value))
	}
	return answer, true
}

// MinimumScore is the lowest title match the model may still pick: a candidate that barely resembles the book's title
// is never taken, however sure the model sounds.
const MinimumScore = 0.3

// Verify returns the zero-based index of the chosen candidate, or false when the model declined, was not confident, named
// a number that is not in the list, or chose a candidate whose title hardly matches.
func Verify(answer Answer, candidates []Candidate) (int, bool) {
	if answer.Confidence != "high" && answer.Confidence != "medium" {
		return 0, false
	}
	if answer.Pick < 1 || answer.Pick > len(candidates) {
		return 0, false
	}
	if candidates[answer.Pick-1].Score < MinimumScore {
		return 0, false
	}
	return answer.Pick - 1, true
}
