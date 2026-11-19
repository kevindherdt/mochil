// Package notepolish prepares a note for an AI to format and polish, and checks the proposal that comes back.
// It never talks to the AI or writes a file: the server asks, the person looks at the proposal, and only then is it saved.
package notepolish

import (
	"regexp"
	"sort"
	"strings"
)

// MaximumBodyBytes keeps one request small enough for a modest local model to answer in full.
const MaximumBodyBytes = 48 << 10

// Instructions is the system message sent with every note.
const Instructions = "You are a careful copy editor for a personal Markdown note. Format and polish the note: " +
	"use a sensible heading hierarchy, consistent bullet and numbered lists, tidy spacing, fenced code blocks (with a language when obvious) for commands and code, " +
	"fix obvious typos and grammar, and group related lines under clear headings. Keep the writer's meaning and voice and the note's language. " +
	"Never add facts and never remove information. Keep every link, [[wikilink]], URL, number, name, API key, token and password exactly, character for character. " +
	"Do not use any tools. Reply with ONLY the complete polished note in Markdown: no introduction, no explanation, and no code fence around the whole note."

var frontMatter = regexp.MustCompile(`^---[ \t]*\r?\n(?s:.*?)\r?\n---[ \t]*(?:\r?\n|$)`)

// SplitFrontMatter separates the leading --- properties block, which is never sent to the AI, from the text.
func SplitFrontMatter(content string) (front, body string) {
	if loc := frontMatter.FindStringIndex(content); loc != nil {
		return content[:loc[1]], content[loc[1]:]
	}
	return "", content
}

// Message is what the AI is asked to polish.
func Message(title, body string) string {
	return "Note title: " + strings.TrimSpace(title) + "\n\n--- NOTE START ---\n" + body + "\n--- NOTE END ---"
}

var wholeFence = regexp.MustCompile("(?s)\\A```[A-Za-z0-9_-]*[ \\t]*\\r?\\n(.*)\\r?\\n```[ \\t]*\\z")

// CleanReply trims the answer and removes a code fence an AI sometimes wraps around the whole note.
func CleanReply(reply string) string {
	reply = strings.TrimSpace(reply)
	if m := wholeFence.FindStringSubmatch(reply); m != nil && strings.Count(reply, "```") == 2 {
		reply = strings.TrimSpace(m[1])
	}
	return reply
}

// Assemble puts the original front matter back in front of the polished text and keeps a final newline when the note had one.
func Assemble(front, originalBody, polished string) string {
	polished = strings.TrimRight(polished, "\r\n")
	if strings.HasSuffix(originalBody, "\n") || originalBody == "" {
		polished += "\n"
	}
	return front + polished
}

var (
	wikilink  = regexp.MustCompile(`\[\[[^\]\n]+\]\]`)
	link      = regexp.MustCompile(`https?://[^\s)>\]"'` + "`" + `]+`)
	longToken = regexp.MustCompile(`[A-Za-z0-9_\-+/=.]{28,}`)
	codeBlock = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")
)

// Lost lists what the original had that the proposal does not: links, wikilinks, long keys or tokens and code blocks.
// Each item is shortened so a secret is never echoed in full. An empty list means everything that matters survived.
func Lost(original, proposal string) []string {
	var lost []string
	check := func(kind string, found []string, keep func(string) bool) {
		seen := map[string]bool{}
		for _, item := range found {
			if seen[item] {
				continue
			}
			seen[item] = true
			if !keep(item) {
				lost = append(lost, kind+": "+short(item))
			}
		}
	}
	contains := func(s string) bool { return strings.Contains(proposal, s) }
	check("link", wikilink.FindAllString(original, -1), contains)
	check("web address", link.FindAllString(original, -1), contains)
	check("key or token", longToken.FindAllString(original, -1), contains)
	var blocks []string
	for _, m := range codeBlock.FindAllStringSubmatch(original, -1) {
		if strings.TrimSpace(m[1]) != "" {
			blocks = append(blocks, strings.TrimRight(m[1], "\n"))
		}
	}
	check("code block", blocks, contains)
	sort.Strings(lost)
	return lost
}

func short(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 12 {
		return string(r[:6]) + "…" + string(r[len(r)-3:])
	}
	return s
}

// Plausible rejects an answer that is empty or far from the original size, which means the AI answered about the note
// instead of returning it.
func Plausible(original, proposal string) bool {
	o, p := len(strings.TrimSpace(original)), len(strings.TrimSpace(proposal))
	if o == 0 {
		return false
	}
	return p*100 >= o*40 && p*100 <= o*250
}
