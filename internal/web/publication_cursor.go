package web

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
)

type publicationCursor struct {
	Kind      string `json:"k"`
	Version   uint64 `json:"v"`
	View      string `json:"w"`
	LibraryID string `json:"l"`
	ItemID    string `json:"i,omitempty"`
}

func publicationView(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)[:12])
}

func encodePublicationCursor(cursor publicationCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodePublicationCursor(value string) (publicationCursor, error) {
	if value == "" {
		return publicationCursor{}, nil
	}
	if len(value) > 1024 {
		return publicationCursor{}, errors.New("cursor is too long")
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return publicationCursor{}, errors.New("cursor is malformed")
	}
	var cursor publicationCursor
	if json.Unmarshal(data, &cursor) != nil || cursor.Version == 0 || cursor.View == "" || cursor.LibraryID == "" || cursor.ItemID == "" || cursor.Kind != "books" && cursor.Kind != "series" && cursor.Kind != "authors" {
		return publicationCursor{}, errors.New("cursor is malformed")
	}
	return cursor, nil
}
