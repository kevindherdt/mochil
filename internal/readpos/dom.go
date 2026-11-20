package readpos

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
)

type node struct {
	name   string // lowercase local name; empty for text
	text   string
	parent *node
	kids   []*node
}

func (n *node) isText() bool { return n.name == "" }

var blockElements = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true, "body": true, "caption": true,
	"dd": true, "div": true, "dl": true, "dt": true, "fieldset": true, "figcaption": true, "figure": true,
	"footer": true, "form": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"header": true, "hr": true, "li": true, "main": true, "nav": true, "ol": true, "p": true, "pre": true,
	"section": true, "table": true, "tbody": true, "td": true, "tfoot": true, "th": true, "thead": true,
	"tr": true, "ul": true,
}

var errNoBody = errors.New("chapter has no body")

// parseChapter builds a tolerant element tree. Real EPUB content is mostly
// well-formed XHTML, but entities and unclosed void elements do occur.
func parseChapter(data []byte) (*node, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.CharsetReader = func(label string, input io.Reader) (io.Reader, error) {
		encoding, err := htmlindex.Get(label)
		if err != nil {
			return nil, err
		}
		return encoding.NewDecoder().Reader(input), nil
	}
	root := &node{name: "#root"}
	current := root
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if root.firstElement("html") != nil {
				break
			}
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			element := &node{name: strings.ToLower(value.Name.Local), parent: current}
			current.kids = append(current.kids, element)
			current = element
		case xml.EndElement:
			name := strings.ToLower(value.Name.Local)
			for open := current; open != root; open = open.parent {
				if open.name == name {
					current = open.parent
					break
				}
			}
		case xml.CharData:
			text := string(value)
			if text == "" {
				continue
			}
			if last := len(current.kids) - 1; last >= 0 && current.kids[last].isText() {
				current.kids[last].text += text
				continue
			}
			current.kids = append(current.kids, &node{text: text, parent: current})
		}
	}
	body := root.find("body")
	if body == nil {
		return nil, errNoBody
	}
	return body, nil
}

func (n *node) firstElement(name string) *node {
	for _, kid := range n.kids {
		if kid.name == name {
			return kid
		}
	}
	return nil
}

func (n *node) find(name string) *node {
	if n.name == name {
		return n
	}
	for _, kid := range n.kids {
		if found := kid.find(name); found != nil {
			return found
		}
	}
	return nil
}
