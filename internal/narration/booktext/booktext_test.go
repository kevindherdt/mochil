package booktext

import (
	"reflect"
	"strings"
	"testing"
)

func cut(text string) []string {
	runes := []rune(text)
	var out []string
	for _, s := range spans(runes) {
		out = append(out, strings.TrimSpace(string(runes[s.from:s.to])))
	}
	return out
}

func texts(sentences []Sentence) []string {
	var out []string
	for _, s := range sentences {
		out = append(out, s.Text)
	}
	return out
}

func chapter(body string) []Sentence {
	return Sentences(strings.NewReader(`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>T. Hidden. Title.</title></head><body>` + body + `</body></html>`))
}

func TestSentencesEndWhereTheyShouldAndKeepClosingQuotes(t *testing.T) {
	got := cut("He ran home quickly. “Where were you?” she asked him. Nobody answered at all.")
	want := []string{"He ran home quickly.", "“Where were you?” she asked him.", "Nobody answered at all."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestAbbreviationsAndInitialsDoNotEndASentence(t *testing.T) {
	got := cut("Dr. Smith met J.R. Tolkien at the old station. They talked for hours.")
	want := []string{"Dr. Smith met J.R. Tolkien at the old station.", "They talked for hours."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestAShortFragmentIsReadWithTheNextSentence(t *testing.T) {
	got := cut("Hello! I came back to see you today.")
	if !reflect.DeepEqual(got, []string{"Hello! I came back to see you today."}) {
		t.Fatalf("got %q", got)
	}
}

func TestAVeryLongSentenceIsReadInPartsAtCommas(t *testing.T) {
	var clauses []string
	for i := 0; i < 30; i++ {
		clauses = append(clauses, "a clause number "+string(rune('a'+i%26))+" of the story")
	}
	long := strings.Join(clauses, ", ") + "."
	parts := cut(long)
	if len(parts) < 2 {
		t.Fatalf("not cut: %d", len(parts))
	}
	for _, p := range parts {
		if len([]rune(p)) > 420 {
			t.Fatalf("part of %d characters", len([]rune(p)))
		}
	}
	if strings.Join(parts, " ") != long {
		t.Fatal("the parts do not add up to the sentence")
	}
}

func TestAChapterIsReadInOrderWithInlineMarkup(t *testing.T) {
	got := texts(chapter(`<h1>Chapter One</h1><p>The lantern swung over the harbour. <em>It was late</em> and the sea was calm.</p>`))
	want := []string{"Chapter One", "The lantern swung over the harbour.", "It was late and the sea was calm."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestScriptsStylesAndRubyAnnotationsAreNotRead(t *testing.T) {
	got := texts(chapter(`<p>Visible words are here today.<ruby>x<rt>furigana</rt></ruby></p><p><style>p { color: red }</style></p><script>var a = "Do not read this.";</script>`))
	if !reflect.DeepEqual(got, []string{"Visible words are here today.x"}) {
		t.Fatalf("got %q", got)
	}
}

func TestTheHeadIsNotRead(t *testing.T) {
	for _, s := range chapter(`<p>Only the body counts here.</p>`) {
		if strings.Contains(s.Text, "Hidden") {
			t.Fatalf("read the title: %q", s.Text)
		}
	}
}

func TestHTMLEntitiesAndNoBreakSpacesAreUnderstood(t *testing.T) {
	got := texts(chapter(`<p>Tom&nbsp;and&nbsp;Jerry &amp; friends&hellip; wait.&nbsp;Then they left for home.</p>`))
	if len(got) != 2 || got[0] != "Tom and Jerry & friends… wait." {
		t.Fatalf("got %q", got)
	}
}

func TestAnEmptyOrBreakOnlyChapterGivesNothing(t *testing.T) {
	if got := chapter(`<p>* * *</p><p>   </p><hr/>`); len(got) != 0 {
		t.Fatalf("got %q", texts(got))
	}
}

func TestBoilerplateSentencesAreLeftOut(t *testing.T) {
	got := texts(chapter(`<p>Copyright © Wilbur Smith 2014</p><p>The lantern swung over the harbour.</p><p>Visit www.example.com for more.</p>`))
	if !reflect.DeepEqual(got, []string{"The lantern swung over the harbour."}) {
		t.Fatalf("got %q", got)
	}
}

func TestStoryTextThatLooksLikeBoilerplateIsKept(t *testing.T) {
	for _, text := range []string{
		"She printed in careful letters, and he read it twice.",
		"He had sent a first edition of the poems to her, signed.",
		"The letter ended there, he thought, at the edge of the page.",
	} {
		if !WorthReading(text) {
			t.Errorf("skipped %q", text)
		}
	}
	for _, text := range []string{"All rights reserved.", "Source ISBN: 9780007535651", "Sign up for the S&S Teen Newsletter.", "First published in Great Britain in 2007"} {
		if WorthReading(text) {
			t.Errorf("read %q", text)
		}
	}
}

func TestMatterTitles(t *testing.T) {
	for _, title := range []string{"Copyright", "Copyright Page", "Table of Contents", "Also by Joe Abercrombie", "About the Author", "Acknowledgements", "Praise for The Subtle Knife", "Sneak Peek"} {
		if !IsMatterTitle(title) {
			t.Errorf("%q should be matter", title)
		}
	}
	for _, title := range []string{"Chapter One", "Prologue", "The Index of Forbidden Books", "Author", ""} {
		if IsMatterTitle(title) {
			t.Errorf("%q should be story", title)
		}
	}
}

func TestSentencesKnowWhereTheyAreAndWhatSurroundsThem(t *testing.T) {
	got := chapter(`<p>When the old captain finally came ashore at dusk, he counted every rope twice before he spoke.</p><p>The end came quietly, as it always does in such stories.</p>`)
	if len(got) != 2 {
		t.Fatalf("got %q", texts(got))
	}
	if got[0].Progression != 0 || got[1].Progression <= 0.5 || got[1].Progression >= 0.7 {
		t.Fatalf("progression %v / %v", got[0].Progression, got[1].Progression)
	}
	if got[0].Lead != "When the old captain finally came ashore at dusk," {
		t.Fatalf("lead %q", got[0].Lead)
	}
	if got[1].Lead != "" {
		t.Fatalf("a sentence this short is not cut: %q", got[1].Lead)
	}
	pair := chapter(`<p>The tide was out and the boats lay on their sides. Nobody on the quay said a word about it.</p>`)
	if len(pair) != 2 || !strings.HasSuffix(pair[1].Before, "boats lay on their sides.") || !strings.HasPrefix(pair[0].After, "Nobody on the quay") || len([]rune(pair[1].Before)) > 40 {
		t.Fatalf("context %q / %q", pair[1].Before, pair[0].After)
	}
}

func TestAChapterThatStopsEarlyIsReadAsFarAsItGoes(t *testing.T) {
	got := Sentences(strings.NewReader(`<html><body><p>The first sentence is complete and fine. The second is cut off <b`))
	if len(got) == 0 || got[0].Text != "The first sentence is complete and fine." {
		t.Fatalf("got %q", texts(got))
	}
}

// Go's \w is ASCII only, so a word filter written as a regexp dropped every
// Chinese, Japanese, Korean, Thai or Cyrillic sentence and read-along produced
// an empty chapter.
func TestNonLatinSentencesSurviveTheWordFilter(t *testing.T) {
	got := cut("今天天气很好。我们一起去公园散步吧。你昨天做了什么？")
	if len(got) != 3 {
		t.Fatalf("CJK sentences were dropped: %q", got)
	}
	if sentences := chapter("<p>今天天气很好。我们一起去公园散步吧。</p>"); len(sentences) != 2 {
		t.Fatalf("CJK chapter produced %d sentences: %q", len(sentences), texts(sentences))
	}
}
