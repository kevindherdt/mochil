package transcode

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Source struct {
	LibraryID string
	ItemID    string
	Revision  string
	Duration  float64
	Kind      string
}

func (s Source) Valid() bool {
	return strings.TrimSpace(s.LibraryID) != "" && strings.TrimSpace(s.ItemID) != "" && strings.TrimSpace(s.Revision) != "" && (s.Kind == "audio" || s.Kind == "video")
}

func Key(source Source, variant string) string {
	if !source.Valid() || strings.TrimSpace(variant) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(source.Kind + "\x00" + source.LibraryID + "\x00" + source.ItemID + "\x00" + source.Revision + "\x00" + variant))
	return hex.EncodeToString(sum[:16])
}
