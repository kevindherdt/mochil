package isbn

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// AskInstructions is the system message for the helper that settles a close call. It is only ever allowed to choose a
// number that is already in the book; Verify checks that again, because a model can invent a plausible number.
const AskInstructions = "You help identify the ISBN of an ebook from numbers found inside its file. " +
	"Choose the ISBN that belongs to THIS book's own edition. Prefer the number on the copyright page; prefer the ebook edition when numbers are labelled; " +
	"never choose one from an \"also by\", next-volume, preview or advertisement list; for a series volume the number must belong to this volume, not another. " +
	"You may answer only with a number from the list or one that is literally printed in the front-matter excerpt. Never invent, correct or complete a number. " +
	"If you cannot tell, answer null. Reply with ONLY this JSON and nothing else: " +
	`{"isbn":"<digits>" or null,"confidence":"high" or "medium" or "low","reason":"<one short sentence>"}`

// Facts are what is already known about the book.
type Facts struct {
	Title   string
	Authors string
	Series  string
	Volume  *float64
	Format  string
}

const snippetRunes = 320

func clip(text string, limit int) string {
	if r := []rune(text); len(r) > limit {
		return string(r[:limit]) + "…"
	}
	return text
}

// AskMessage lists the book and every number found, each with the reasons it ranks where it does and the words around it.
func AskMessage(facts Facts, result Result, evidence Evidence) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Book: %s\n", facts.Title)
	if facts.Authors != "" {
		fmt.Fprintf(&b, "Author: %s\n", facts.Authors)
	}
	if facts.Series != "" {
		fmt.Fprintf(&b, "Series: %s", facts.Series)
		if facts.Volume != nil {
			fmt.Fprintf(&b, ", volume %s", strconv.FormatFloat(*facts.Volume, 'f', -1, 64))
		}
		b.WriteString("\n")
	}
	if facts.Format != "" {
		fmt.Fprintf(&b, "Format: %s\n", facts.Format)
	}
	if len(result.Candidates) == 0 {
		b.WriteString("\nNo ISBN-like number was found by the automatic search.\n")
	} else {
		b.WriteString("\nNumbers found in the book (best guess first):\n")
		for index, c := range result.Candidates {
			fmt.Fprintf(&b, "%d. %s — %s\n", index+1, c.ISBN, strings.Join(c.Reasons, ", "))
			if text := evidence.Snippets[c.ISBN]; text != "" {
				fmt.Fprintf(&b, "   text around it: \"%s\"\n", clip(text, snippetRunes))
			}
		}
	}
	if evidence.FrontMatter != "" {
		fmt.Fprintf(&b, "\nFront-matter excerpt:\n%s\n", evidence.FrontMatter)
	}
	return b.String()
}

// Answer is what the helper replied.
type Answer struct {
	ISBN       string `json:"isbn"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
}

// ParseAnswer reads the JSON object out of a reply that may have words or a code fence around it.
func ParseAnswer(reply string) (Answer, bool) {
	start, end := strings.Index(reply, "{"), strings.LastIndex(reply, "}")
	if start < 0 || end <= start {
		return Answer{}, false
	}
	var raw struct {
		ISBN       any    `json:"isbn"`
		Confidence string `json:"confidence"`
		Reason     string `json:"reason"`
	}
	if json.Unmarshal([]byte(reply[start:end+1]), &raw) != nil {
		return Answer{}, false
	}
	answer := Answer{Confidence: strings.ToLower(strings.TrimSpace(raw.Confidence)), Reason: clip(strings.TrimSpace(raw.Reason), 240)}
	switch value := raw.ISBN.(type) {
	case string:
		answer.ISBN = value
	case float64:
		answer.ISBN = strconv.FormatFloat(value, 'f', 0, 64)
	}
	return answer, true
}

// Verify accepts the helper's choice only when it is a correct ISBN that is really in the book: one of the numbers the
// search found, or one printed in the front-matter excerpt. It returns the ISBN-13, or false when the helper declined
// or named a number that is not there.
func Verify(answer Answer, result Result, evidence Evidence) (string, bool) {
	isbn, ok := Normalize(answer.ISBN)
	if !ok {
		return "", false
	}
	for _, c := range result.Candidates {
		if c.ISBN == isbn {
			return isbn, true
		}
	}
	for _, printed := range FindIn(evidence.FrontMatter) {
		if printed == isbn {
			return isbn, true
		}
	}
	return "", false
}

// Choose records a verified answer: the number becomes the book's own, first in the list, with the helper's reason.
func Choose(result Result, isbn string, answer Answer) Result {
	reason := "picked by Hermes (" + answer.Confidence + " confidence)"
	if answer.Reason != "" {
		reason += ": " + answer.Reason
	}
	chosen := Candidate{ISBN: isbn, Reasons: []string{reason}}
	rest := make([]Candidate, 0, len(result.Candidates))
	for _, c := range result.Candidates {
		if c.ISBN == isbn {
			chosen.Score = c.Score
			chosen.Reasons = append(chosen.Reasons, c.Reasons...)
			continue
		}
		rest = append(rest, c)
	}
	return Result{Own: isbn, Candidates: append([]Candidate{chosen}, rest...)}
}
