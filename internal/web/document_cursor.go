package web

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

type documentCursor struct {
	Version uint64 `json:"v"`
	Sort    string `json:"s"`
	ItemID  string `json:"i"`
}

func encodeDocumentCursor(cursor documentCursor) string {
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeDocumentCursor(value string) (documentCursor, error) {
	if value == "" {
		return documentCursor{}, nil
	}
	if len(value) > 512 {
		return documentCursor{}, errors.New("cursor is too long")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return documentCursor{}, errors.New("cursor is malformed")
	}
	var cursor documentCursor
	if json.Unmarshal(decoded, &cursor) != nil || cursor.Version == 0 || cursor.ItemID == "" || (cursor.Sort != "title" && cursor.Sort != "recent") {
		return documentCursor{}, errors.New("cursor is malformed")
	}
	return cursor, nil
}
