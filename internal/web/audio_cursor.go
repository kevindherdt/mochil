package web

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

type audioCursor struct {
	AudioVersion    uint64 `json:"a"`
	PodcastVersion  uint64 `json:"p,omitempty"`
	ProgressVersion uint64 `json:"r,omitempty"`
	View            string `json:"v"`
	ItemID          string `json:"i"`
}

func audioViewKey(libraryID, query, show, group, sortOrder, played string) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{libraryID, query, show, group, sortOrder, played}, "\x00")))
	return hex.EncodeToString(digest[:12])
}

func encodeAudioCursor(cursor audioCursor) string {
	encoded, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeAudioCursor(value string) (audioCursor, error) {
	if value == "" {
		return audioCursor{}, nil
	}
	if len(value) > 512 {
		return audioCursor{}, errors.New("cursor is too long")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) > 384 {
		return audioCursor{}, errors.New("cursor is malformed")
	}
	var cursor audioCursor
	if json.Unmarshal(decoded, &cursor) != nil || cursor.AudioVersion == 0 || cursor.View == "" || cursor.ItemID == "" {
		return audioCursor{}, errors.New("cursor is malformed")
	}
	return cursor, nil
}
