package booktext

import (
	"encoding/xml"
	"io"
	"strings"
	"unicode"
)

var blocks = map[string]bool{
	"P": true, "DIV": true, "LI": true, "H1": true, "H2": true, "H3": true, "H4": true, "H5": true, "H6": true,
	"BLOCKQUOTE": true, "TD": true, "TH": true, "DT": true, "DD": true, "SECTION": true, "ARTICLE": true,
	"FIGCAPTION": true, "PRE": true, "BODY": true,
}

var skipped = map[string]bool{
	"SCRIPT": true, "STYLE": true, "RT": true, "RP": true, "NOSCRIPT": true, "SVG": true, "MATH": true,
}

// Sentence is one stretch of a chapter to be read aloud.
type Sentence struct {
	Text string `json:"text"`
	// Lead is the opening clause of a long sentence. Speech can start as soon as it is
	// ready, then carry on with the rest. Empty when the sentence is short.
	Lead string `json:"lead,omitempty"`
	// Progression is how far into the chapter's text the sentence starts, from 0 to 1.
	Progression float64 `json:"progression"`
	// Before and After are the words around the sentence, for finding it again in the page.
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type group struct {
	text   []rune
	offset int // characters of chapter text before this group
}

const context = 40

// Sentences reads a chapter's XHTML and returns what is worth reading aloud, in order.
// It follows the same rules as the web reader: text inside scripts, ruby annotations and
// the like is skipped, and each block of text is cut into sentences on its own.
func Sentences(chapter io.Reader) []Sentence {
	decoder := xml.NewDecoder(chapter)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }

	type frame struct {
		skipped bool
		block   int // id of the nearest enclosing block, 0 for none
	}
	var (
		stack   []frame
		inBody  int
		nextID  int
		groups  []*group
		current *group
		last    = -1
		total   int
	)
	for {
		token, err := decoder.Token()
		if err != nil {
			break // a chapter that stops early is read as far as it goes
		}
		switch t := token.(type) {
		case xml.StartElement:
			name := strings.ToUpper(t.Name.Local)
			parent := frame{}
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			next := frame{skipped: parent.skipped || skipped[name], block: parent.block}
			if blocks[name] {
				nextID++
				next.block = nextID
			}
			if name == "BODY" {
				inBody++
			}
			stack = append(stack, next)
		case xml.EndElement:
			if len(stack) == 0 {
				continue
			}
			if strings.EqualFold(t.Name.Local, "body") && inBody > 0 {
				inBody--
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if inBody == 0 || len(stack) == 0 || len(t) == 0 {
				continue
			}
			top := stack[len(stack)-1]
			if top.skipped {
				continue
			}
			if current == nil || last != top.block {
				current = &group{offset: total}
				groups = append(groups, current)
				last = top.block
			}
			runes := []rune(string(t))
			current.text = append(current.text, runes...)
			total += len(runes)
		}
	}

	var out []Sentence
	for _, g := range groups {
		for _, s := range spans(g.text) {
			start, end := s.from, s.to
			for start < end && isSpace(g.text[start]) {
				start++
			}
			for end > start && isSpace(g.text[end-1]) {
				end--
			}
			text := strings.TrimSpace(runsOfSpace.ReplaceAllString(string(g.text[start:end]), " "))
			if !hasWord(text) || !WorthReading(text) {
				continue
			}
			sentence := Sentence{Text: text}
			if total > 0 {
				sentence.Progression = float64(g.offset+start) / float64(total)
			}
			if cut := leadEnd(g.text[start:end]); cut > 0 {
				sentence.Lead = strings.TrimSpace(runsOfSpace.ReplaceAllString(string(g.text[start:start+cut]), " "))
			}
			sentence.Before = tidy(g.text[max(0, start-context):start])
			sentence.After = tidy(g.text[end:min(len(g.text), end+context)])
			out = append(out, sentence)
		}
	}
	return out
}

func isSpace(r rune) bool { return unicode.IsSpace(r) || r == 0xfeff }

func tidy(runes []rune) string {
	return strings.TrimSpace(runsOfSpace.ReplaceAllString(string(runes), " "))
}
