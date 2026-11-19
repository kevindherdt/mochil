package document

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

const maximumOfficeContainer = int64(64 << 20)

func extractDOCX(reader io.ReaderAt, size, maxBytes int64) ([]byte, error) {
	return extractOfficeXML(reader, size, maxBytes, "word/document.xml", decodeWordprocessingML)
}

func extractOfficeXML(reader io.ReaderAt, size, maxBytes int64, name string, decode func(io.Reader, int64) ([]byte, error)) ([]byte, error) {
	if size > maximumOfficeContainer {
		return nil, ErrTextTooLarge
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, ErrTextMalformed
	}
	var document *zip.File
	for _, entry := range archive.File {
		if entry.Name == name {
			document = entry
			break
		}
	}
	if document == nil {
		return nil, ErrTextMalformed
	}
	maximumXML := uint64(16 << 20)
	if maxBytes < 2<<20 {
		maximumXML = uint64(maxBytes * 8)
	}
	if document.UncompressedSize64 > maximumXML {
		return nil, ErrTextTooLarge
	}
	content, err := document.Open()
	if err != nil {
		return nil, ErrTextMalformed
	}
	defer content.Close()
	return decode(io.LimitReader(content, int64(maximumXML)+1), maxBytes)
}

func decodeWordprocessingML(reader io.Reader, maxBytes int64) ([]byte, error) {
	decoder := xml.NewDecoder(reader)
	var output strings.Builder
	inText := false
	appendText := func(value string) error {
		if int64(output.Len()+len(value)) > maxBytes {
			return ErrTextTooLarge
		}
		output.WriteString(value)
		return nil
	}
	// Word writes a text box or drawing twice: once for a modern renderer inside
	// mc:Choice and once for a legacy one inside mc:Fallback. Both copies carry the
	// same text, so the fallback subtree is skipped rather than read twice.
	skipDepth := 0
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
			if skipDepth > 0 {
				skipDepth++
				continue
			}
			if value.Name.Local == "Fallback" {
				skipDepth = 1
				continue
			}
			switch value.Name.Local {
			case "t":
				inText = true
			case "tab":
				if err := appendText("\t"); err != nil {
					return nil, err
				}
			case "br", "cr":
				if err := appendText("\n"); err != nil {
					return nil, err
				}
			}
		case xml.CharData:
			if inText {
				if err := appendText(string(value)); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if skipDepth > 0 {
				skipDepth--
				continue
			}
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				if output.Len() > 0 {
					if err := appendText("\n"); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return []byte(strings.TrimSpace(output.String())), nil
}
