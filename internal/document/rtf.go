package document

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

type rtfState struct {
	ignored bool
	uc      int
	skip    int
	charset *charmap.Charmap // declared by \ansicpg; nil means Windows-1252
}

// rtfCodePages are the Windows code pages RTF documents name with \ansicpg.
var rtfCodePages = map[int]*charmap.Charmap{
	1250: charmap.Windows1250, 1251: charmap.Windows1251, 1252: charmap.Windows1252,
	1253: charmap.Windows1253, 1254: charmap.Windows1254, 1255: charmap.Windows1255,
	1256: charmap.Windows1256, 1257: charmap.Windows1257, 1258: charmap.Windows1258,
}

// decodeByte reads one 8-bit character in the document's code page.
func (s *rtfState) decodeByte(value byte) rune {
	if s.charset != nil {
		return s.charset.DecodeByte(value)
	}
	return windows1252(value)
}

var ignoredRTFDestinations = map[string]bool{
	"annotation": true, "colortbl": true, "datafield": true, "filetbl": true,
	"fonttbl": true, "footer": true, "footerf": true, "footerl": true,
	"footerr": true, "footnote": true, "generator": true, "header": true,
	"headerf": true, "headerl": true, "headerr": true, "info": true,
	"listoverridetable": true, "listtable": true, "object": true, "pict": true,
	"stylesheet": true,
}

func decodeRTF(input []byte, maxBytes int64) ([]byte, error) {
	if len(input) < 6 || !strings.HasPrefix(string(input[:min(len(input), 6)]), "{\\rtf") {
		return nil, ErrTextMalformed
	}
	states := []rtfState{{uc: 1}}
	var output strings.Builder
	writeRune := func(value rune) error {
		if int64(output.Len()+utf8.RuneLen(value)) > maxBytes {
			return ErrTextTooLarge
		}
		output.WriteRune(value)
		return nil
	}
	// RTF has no code point above U+FFFF, so a supplementary character arrives as a
	// UTF-16 surrogate pair in two \u words. Writing either half alone is invalid, so
	// hold a high surrogate until its partner (or the next unit) shows whether they pair.
	pendingHigh := rune(-1)
	appendRune := func(value rune) error {
		if pendingHigh >= 0 {
			high, rest := pendingHigh, value
			pendingHigh = -1
			if rest >= 0xDC00 && rest <= 0xDFFF {
				return writeRune(0x10000 + (high-0xD800)<<10 + (rest - 0xDC00))
			}
			if err := writeRune(utf8.RuneError); err != nil {
				return err
			}
			value = rest
		}
		if value >= 0xD800 && value <= 0xDBFF {
			pendingHigh = value
			return nil
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			return writeRune(utf8.RuneError)
		}
		return writeRune(value)
	}
	for index := 0; index < len(input); {
		state := &states[len(states)-1]
		switch input[index] {
		case '{':
			if len(states) >= 256 {
				return nil, ErrTextMalformed
			}
			states = append(states, *state)
			index++
		case '}':
			if len(states) == 1 {
				return nil, ErrTextMalformed
			}
			states = states[:len(states)-1]
			index++
		case '\\':
			index++
			if index >= len(input) {
				return nil, ErrTextMalformed
			}
			state = &states[len(states)-1]
			symbol := input[index]
			if symbol == '\\' || symbol == '{' || symbol == '}' {
				index++
				if state.skip > 0 {
					state.skip--
					continue
				}
				if !state.ignored {
					if err := appendRune(rune(symbol)); err != nil {
						return nil, err
					}
				}
				continue
			}
			if symbol == '\'' {
				if index+2 >= len(input) {
					return nil, ErrTextMalformed
				}
				value, err := strconv.ParseUint(string(input[index+1:index+3]), 16, 8)
				if err != nil {
					return nil, ErrTextMalformed
				}
				index += 3
				if state.skip > 0 {
					state.skip--
					continue
				}
				if !state.ignored {
					if err := appendRune(state.decodeByte(byte(value))); err != nil {
						return nil, err
					}
				}
				continue
			}
			if !unicode.IsLetter(rune(symbol)) {
				index++
				if symbol == '*' {
					state.ignored = true
					continue
				}
				if state.skip > 0 {
					state.skip--
					continue
				}
				if !state.ignored && (symbol == '~' || symbol == '_') {
					value := ' '
					if symbol == '_' {
						value = '-'
					}
					if err := appendRune(value); err != nil {
						return nil, err
					}
				}
				continue
			}
			start := index
			for index < len(input) && unicode.IsLetter(rune(input[index])) {
				index++
			}
			word := string(input[start:index])
			numberStart := index
			if index < len(input) && (input[index] == '-' || input[index] == '+') {
				index++
			}
			for index < len(input) && input[index] >= '0' && input[index] <= '9' {
				index++
			}
			numberText := string(input[numberStart:index])
			if index < len(input) && input[index] == ' ' {
				index++
			}
			if word == "bin" {
				// \binN is followed by exactly N raw bytes, braces included; they are
				// never syntax, and skipping them is required even inside a group we drop.
				count, err := strconv.Atoi(numberText)
				// Compare against what is left rather than adding to index: a huge
				// count overflowed the addition and panicked with a negative index.
				if err != nil || count < 0 || count > len(input)-index {
					return nil, ErrTextMalformed
				}
				index += count
				continue
			}
			if ignoredRTFDestinations[word] {
				state.ignored = true
			}
			if state.ignored {
				continue
			}
			switch word {
			case "par", "line":
				if err := appendRune('\n'); err != nil {
					return nil, err
				}
			case "tab":
				if err := appendRune('\t'); err != nil {
					return nil, err
				}
			case "ansicpg":
				if page, err := strconv.Atoi(numberText); err == nil {
					if table, ok := rtfCodePages[page]; ok {
						state.charset = table
					}
				}
			case "uc":
				if value, err := strconv.Atoi(numberText); err == nil && value >= 0 && value <= 16 {
					state.uc = value
				}
			case "emdash":
				if err := appendRune('—'); err != nil {
					return nil, err
				}
			case "endash":
				if err := appendRune('–'); err != nil {
					return nil, err
				}
			case "bullet":
				if err := appendRune('•'); err != nil {
					return nil, err
				}
			case "lquote":
				if err := appendRune('‘'); err != nil {
					return nil, err
				}
			case "rquote":
				if err := appendRune('’'); err != nil {
					return nil, err
				}
			case "ldblquote":
				if err := appendRune('“'); err != nil {
					return nil, err
				}
			case "rdblquote":
				if err := appendRune('”'); err != nil {
					return nil, err
				}
			case "emspace", "enspace", "qmspace":
				if err := appendRune(' '); err != nil {
					return nil, err
				}
			case "u":
				value, err := strconv.Atoi(numberText)
				if err != nil {
					return nil, ErrTextMalformed
				}
				if value < 0 {
					value += 65536
				}
				if err := appendRune(rune(value)); err != nil {
					return nil, err
				}
				state.skip = state.uc
			}
		default:
			value := input[index]
			index++
			if value == '\r' || value == '\n' {
				continue
			}
			if state.skip > 0 {
				state.skip--
				continue
			}
			if !state.ignored {
				if err := appendRune(state.decodeByte(value)); err != nil {
					return nil, err
				}
			}
		}
	}
	if len(states) != 1 {
		return nil, ErrTextMalformed
	}
	if pendingHigh >= 0 {
		// A document ending on a high surrogate has no partner left to pair with.
		if err := writeRune(utf8.RuneError); err != nil {
			return nil, err
		}
	}
	return []byte(strings.TrimSpace(output.String())), nil
}

func windows1252(value byte) rune {
	if value < 0x80 || value >= 0xa0 {
		return rune(value)
	}
	const replacements = "€�‚ƒ„…†‡ˆ‰Š‹Œ�Ž��‘’“”•–—˜™š›œ�žŸ"
	return []rune(replacements)[value-0x80]
}
