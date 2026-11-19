package notepolish

import (
	"strings"
	"testing"
)

func TestSplitFrontMatterKeepsPropertiesOutOfTheText(t *testing.T) {
	front, body := SplitFrontMatter("---\ntags: [a]\n---\n# Title\ntext\n")
	if front != "---\ntags: [a]\n---\n" || body != "# Title\ntext\n" {
		t.Fatalf("front=%q body=%q", front, body)
	}
	if front, body := SplitFrontMatter("no properties\n---\nnot front matter\n"); front != "" || body == "" {
		t.Fatalf("a rule in the middle is not front matter: %q %q", front, body)
	}
	if front, _ := SplitFrontMatter("---\r\na: 1\r\n---\r\ntext"); front != "---\r\na: 1\r\n---\r\n" {
		t.Fatalf("CRLF front matter = %q", front)
	}
}

func TestCleanReplyRemovesAWrappingFenceOnly(t *testing.T) {
	if got := CleanReply("```markdown\n# A\n\ntext\n```\n"); got != "# A\n\ntext" {
		t.Fatalf("got %q", got)
	}
	inner := "# A\n\n```sh\nls\n```\n"
	if got := CleanReply(inner); got != strings.TrimSpace(inner) {
		t.Fatalf("a note that merely contains a code block must be left alone: %q", got)
	}
}

func TestAssembleRestoresFrontMatterAndTrailingNewline(t *testing.T) {
	if got := Assemble("---\na: 1\n---\n", "old\n", "new\n\n"); got != "---\na: 1\n---\nnew\n" {
		t.Fatalf("got %q", got)
	}
	if got := Assemble("", "old", "new"); got != "new" {
		t.Fatalf("no newline in the original, none added: %q", got)
	}
}

func TestLostReportsMissingLinksTokensAndCodeWithoutEchoingSecrets(t *testing.T) {
	secret := "eyJhbGciOiJIUzI1NiJ9.abcdefghijklmnopqrstuvwxyz0123"
	original := "See [[Server setup]] and https://example.com/a?b=1\n\nkey " + secret + "\n\n```sh\nsudo systemctl restart x\n```\n"
	good := "# Notes\n\n- See [[Server setup]] and https://example.com/a?b=1\n- key " + secret + "\n\n```sh\nsudo systemctl restart x\n```\n"
	if lost := Lost(original, good); len(lost) != 0 {
		t.Fatalf("nothing was lost: %v", lost)
	}
	bad := "# Notes\n\nSee Server setup at example.com. key eyJhbGci...\n\n```sh\nrestart x\n```\n"
	lost := strings.Join(Lost(original, bad), "\n")
	for _, want := range []string{"link: [[Serv", "web address:", "key or token:", "code block:"} {
		if !strings.Contains(lost, want) {
			t.Fatalf("missing %q in %q", want, lost)
		}
	}
	if strings.Contains(lost, secret) || strings.Contains(lost, "abcdefghijklmnop") {
		t.Fatalf("a secret was echoed: %q", lost)
	}
}

func TestPlausibleRejectsEmptyAndWildlyDifferentAnswers(t *testing.T) {
	original := strings.Repeat("word ", 100)
	for proposal, want := range map[string]bool{"": false, "Sorry, I cannot help": false, original + original + original: false, strings.Repeat("word ", 90): true} {
		if got := Plausible(original, proposal); got != want {
			t.Fatalf("Plausible(%q…) = %v, want %v", proposal[:min(len(proposal), 10)], got, want)
		}
	}
}
