package document

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestWikiLinkCodeAndAliasRegressions(t *testing.T) {
	cases := []struct {
		name, text string
		want       []string
	}{
		{"unmatched backtick", "before ` [[Good]]", []string{"good"}},
		{"escaped backtick", "before \\` [[Good]]", []string{"good"}},
		{"escaped first of two opens span", "\\`` [[Hidden]] ` [[Good]]", []string{"good"}},
		{"escaped first of three opens span", "\\``` [[Hidden]] `` [[Good]]", []string{"good"}},
		{"escaped first of closing run remains code", "`` [[Hidden]] \\`` [[Good]]", []string{"good"}},
		{"backtick in fence info", "```bad`info\n[[Good]]\n```", []string{"good"}},
		{"tilde fence accepts backtick info", "~~~has`backtick\n[[Hidden]]\n~~~\n[[Good]]", []string{"good"}},
		{"valid backtick fence", "```lang\n[[Hidden]]\n```\n[[Good]]", []string{"good"}},
		{"even backslashes do not escape", "\\\\`` [[Hidden]] `` [[Good]]", []string{"good"}},
		{"multiple backticks", "`` ` [[Hidden]] ` `` [[Good]]", []string{"good"}},
		{"multiline code", "`code\n[[Hidden]]` [[Good]]", []string{"good"}},
		{"no join across code", "[[Tar`code`get]] [[Good]]", []string{"good"}},
		{"alias opening bracket", "[[Target|label [nested]]", []string{"target"}},
		{"alias nested opener", "[[Target|label [[nested]]", []string{"target"}},
		{"blank line ends code span", "`unmatched\n \n[[Good]]`", []string{"good"}},
		{"nested openers", "[[broken [[also [[Good]]", []string{"good"}},
		{"line boundary", "[[Broken\nlink]] [[Good]]", []string{"good"}},
		{"unmatched backtick before later code", "`` unmatched `[[Hidden]]` [[Good]]", []string{"good"}},
		{"escaped closer inside code", "`hidden \\` [[Good]]", []string{"good"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wikiLinkTargets(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestWikiLinkBacktickHeavyAllocation(t *testing.T) {
	text := strings.Repeat("`x", 1<<20)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	wikiLinkTargets(text)
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 40<<20 {
		t.Fatalf("allocated %d bytes for %d-byte input, want <= 40 MiB", allocated, len(text))
	}
}

func BenchmarkWikiLinkBacktickHeavy(b *testing.B) {
	text := strings.Repeat("`x", 1<<20)
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wikiLinkTargets(text)
	}
}

func BenchmarkWikiLinkNestedOpeners(b *testing.B) {
	for _, n := range []int{1024, 8192, 65536} {
		text := strings.Repeat("[[x", n) + "]]"
		b.Run(strings.Repeat("x", 0)+fmtSize(n), func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			for i := 0; i < b.N; i++ {
				wikiLinkTargets(text)
			}
		})
	}
}

func fmtSize(n int) string {
	switch n {
	case 1024:
		return "1K"
	case 8192:
		return "8K"
	default:
		return "64K"
	}
}
