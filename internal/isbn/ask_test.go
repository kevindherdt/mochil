package isbn

import (
	"strings"
	"testing"
)

func TestEvidenceKeepsTheTextAroundEachNumberAndTheCopyrightPage(t *testing.T) {
	f := book(t, nil, [][2]string{
		{"title.xhtml", "<p>A Tale</p>"},
		{"copyright.xhtml", "<p>Copyright 2020 Example Press. ISBN 978-0-306-40615-7 (ebook). Printed in Japan.</p>"},
		{"ads.xhtml", "<p>Also by this author: Another Tale ISBN 978-0-8044-2957-3</p>"},
	}, map[string]string{"copyright.xhtml": "copyright-page"})
	result, evidence, err := ExtractEvidence(f, "epub", Hints{Title: "A Tale"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates = %+v", result.Candidates)
	}
	if !strings.Contains(evidence.Snippets["9780306406157"], "Example Press") {
		t.Fatalf("snippet = %q", evidence.Snippets["9780306406157"])
	}
	if !strings.Contains(evidence.FrontMatter, "Printed in Japan") || strings.Contains(evidence.FrontMatter, "Another Tale") {
		t.Fatalf("front matter = %q", evidence.FrontMatter)
	}
}

func TestAskMessageListsTheBookAndEveryNumberWithItsWords(t *testing.T) {
	result := Result{Candidates: []Candidate{{ISBN: "9780306406157", Score: 40, Reasons: []string{"on the copyright page"}}}}
	message := AskMessage(Facts{Title: "A Tale", Authors: "Ann", Series: "Tales", Volume: volume(2), Format: "epub"}, result,
		Evidence{Snippets: map[string]string{"9780306406157": "Copyright Example Press"}, FrontMatter: "Printed in Japan"})
	for _, want := range []string{"Book: A Tale", "Author: Ann", "Series: Tales, volume 2", "9780306406157 — on the copyright page", "Copyright Example Press", "Printed in Japan"} {
		if !strings.Contains(message, want) {
			t.Fatalf("missing %q in\n%s", want, message)
		}
	}
}

func TestParseAnswerReadsJSONAroundWordsAndFences(t *testing.T) {
	for reply, want := range map[string]string{
		`{"isbn":"978-0-306-40615-7","confidence":"High","reason":"copyright page"}`:  "978-0-306-40615-7",
		"Sure!\n```json\n{\"isbn\": \"9780306406157\", \"confidence\": \"low\"}\n```": "9780306406157",
		`{"isbn":null,"confidence":"low","reason":"cannot tell"}`:                     "",
		`{"isbn":9780306406157,"confidence":"high","reason":"number not string"}`:     "9780306406157",
	} {
		got, ok := ParseAnswer(reply)
		if !ok || got.ISBN != want {
			t.Errorf("ParseAnswer(%q) = %+v %v, want ISBN %q", reply, got, ok, want)
		}
	}
	if _, ok := ParseAnswer("I think it is the first one."); ok {
		t.Fatal("prose without JSON is not an answer")
	}
}

func TestVerifyAcceptsOnlyNumbersReallyInTheBook(t *testing.T) {
	result := Result{Candidates: []Candidate{{ISBN: "9780306406157"}}}
	evidence := Evidence{FrontMatter: "Digital edition ISBN 978-0-8044-2957-3 published 2020"}
	if got, ok := Verify(Answer{ISBN: "0-306-40615-2"}, result, evidence); !ok || got != "9780306406157" {
		t.Fatalf("a candidate in ISBN-10 form: %q %v", got, ok)
	}
	if got, ok := Verify(Answer{ISBN: "9780804429573"}, result, evidence); !ok || got != "9780804429573" {
		t.Fatalf("a number printed in the front matter: %q %v", got, ok)
	}
	// A correct ISBN that is not in the book is an invention, however plausible.
	if _, ok := Verify(Answer{ISBN: "9780141036144"}, result, evidence); ok {
		t.Fatal("an ISBN that is not in the book must be refused")
	}
	if _, ok := Verify(Answer{ISBN: "9780306406158"}, result, evidence); ok {
		t.Fatal("a wrong check digit must be refused")
	}
	if _, ok := Verify(Answer{}, result, evidence); ok {
		t.Fatal("declining is not a choice")
	}
}

func TestChoosePutsTheVerifiedNumberFirstWithTheHelpersReason(t *testing.T) {
	result := Result{Ambiguous: true, Candidates: []Candidate{{ISBN: "9780804429573", Score: 50}, {ISBN: "9780306406157", Score: 40, Reasons: []string{"on the copyright page"}}}}
	got := Choose(result, "9780306406157", Answer{Confidence: "high", Reason: "it is on the copyright page"})
	if got.Own != "9780306406157" || got.Ambiguous || len(got.Candidates) != 2 || got.Candidates[0].ISBN != "9780306406157" || got.Candidates[1].ISBN != "9780804429573" {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(got.Candidates[0].Reasons[0], "picked by Hermes (high confidence): it is on the copyright page") || got.Candidates[0].Reasons[1] != "on the copyright page" {
		t.Fatalf("reasons = %v", got.Candidates[0].Reasons)
	}
}
