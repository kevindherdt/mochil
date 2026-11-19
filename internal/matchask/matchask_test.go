package matchask

import (
	"strings"
	"testing"
)

var candidates = []Candidate{
	{Title: "Dune", Authors: "Frank Herbert", Year: 1965, Score: 0.95},
	{Title: "Dune Messiah", Authors: "Frank Herbert", Year: 1969, Score: 0.6},
	{Title: "Dune: A Study Guide", Authors: "Someone Else", Score: 0.2},
}

func TestMessageNumbersTheCandidatesAndKeepsTheBookFacts(t *testing.T) {
	volume := 1.0
	text := Message(Book{Title: "Dune", Authors: "Frank Herbert", Series: "Dune", Volume: &volume, Description: strings.Repeat("sand ", 200)}, candidates)
	for _, want := range []string{"Library book: Dune", "Series: Dune, volume 1", "1. Dune — by Frank Herbert (1965) [title match 95%]", "2. Dune Messiah", "3. Dune: A Study Guide"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if strings.Count(text, "sand") > 70 {
		t.Fatal("the description must be clipped")
	}
}

func TestParseAnswerAcceptsFencesNumbersAndStrings(t *testing.T) {
	for reply, pick := range map[string]int{
		"```json\n{\"pick\":2,\"confidence\":\"High\",\"reason\":\"same\"}\n```": 2,
		`Sure: {"pick":"3","confidence":"medium","reason":"x"}`:                  3,
		`{"pick":null,"confidence":"low","reason":"unsure"}`:                     0,
	} {
		answer, ok := ParseAnswer(reply)
		if !ok || answer.Pick != pick {
			t.Fatalf("%q = %+v %v", reply, answer, ok)
		}
	}
	if _, ok := ParseAnswer("I think the first one."); ok {
		t.Fatal("prose is not an answer")
	}
}

func TestVerifyOnlyTakesAConfidentListedCandidateThatResemblesTheTitle(t *testing.T) {
	cases := []struct {
		answer Answer
		index  int
		ok     bool
	}{
		{Answer{Pick: 1, Confidence: "high"}, 0, true},
		{Answer{Pick: 2, Confidence: "medium"}, 1, true},
		{Answer{Pick: 1, Confidence: "low"}, 0, false},
		{Answer{Pick: 0, Confidence: "high"}, 0, false},
		{Answer{Pick: 4, Confidence: "high"}, 0, false},
		{Answer{Pick: -1, Confidence: "high"}, 0, false},
		{Answer{Pick: 3, Confidence: "high"}, 0, false}, // title match 20%: never
	}
	for _, c := range cases {
		if index, ok := Verify(c.answer, candidates); ok != c.ok || (ok && index != c.index) {
			t.Fatalf("%+v = %d %v", c.answer, index, ok)
		}
	}
}
