package document

import (
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
)

// maximumODTSpaces bounds one text:s repeat so a crafted count cannot balloon the output.
const maximumODTSpaces = 1024

func extractODT(reader io.ReaderAt, size, maxBytes int64) ([]byte, error) {
	return extractOfficeXML(reader, size, maxBytes, "content.xml", decodeOpenDocumentText)
}

func decodeOpenDocumentText(reader io.Reader, maxBytes int64) ([]byte, error) {
	decoder := xml.NewDecoder(reader)
	var output strings.Builder
	textDepth := 0
	appendText := func(value string) error {
		if int64(output.Len()+len(value)) > maxBytes {
			return ErrTextTooLarge
		}
		output.WriteString(value)
		return nil
	}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrTextMalformed
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "p", "h", "span", "a":
				textDepth++
			case "tab":
				if textDepth > 0 {
					if err := appendText("\t"); err != nil {
						return nil, err
					}
				}
			case "line-break":
				if textDepth > 0 {
					if err := appendText("\n"); err != nil {
						return nil, err
					}
				}
			case "s":
				if textDepth > 0 {
					// Text:s carries how many spaces it stands for; the default is one.
					count := 1
					for _, attr := range value.Attr {
						if attr.Name.Local == "c" {
							if parsed, err := strconv.Atoi(attr.Value); err == nil && parsed > 0 && parsed <= maximumODTSpaces {
								count = parsed
							}
						}
					}
					if err := appendText(strings.Repeat(" ", count)); err != nil {
						return nil, err
					}
				}
			}
		case xml.CharData:
			if textDepth > 0 {
				if err := appendText(string(value)); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "p", "h":
				textDepth--
				if output.Len() > 0 {
					if err := appendText("\n"); err != nil {
						return nil, err
					}
				}
			case "span", "a":
				textDepth--
			}
		}
	}
	return []byte(strings.TrimSpace(output.String())), nil
}
