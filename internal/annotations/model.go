package annotations

import (
	"math"
	"strings"
	"unicode/utf8"

	"mochil/internal/reading"
)

const (
	MaximumIDBytes       = 128
	MaximumQuoteBytes    = 8192
	MaximumNoteBytes     = 16384
	MaximumColorBytes    = 32
	MaximumChapterBytes  = 1024
	MaximumItemsPerBook  = 10000
	MaximumItemsPerBatch = 500
)

type Annotation struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Quote     string            `json:"quote,omitempty"`
	Note      string            `json:"note,omitempty"`
	Color     string            `json:"color,omitempty"`
	Chapter   string            `json:"chapter,omitempty"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
	Revision  int64             `json:"revision"`
	Deleted   bool              `json:"deleted,omitempty"`
	DeviceID  string            `json:"device_id,omitempty"`
	Anchor    *reading.Anchor   `json:"anchor,omitempty"`
	Locators  map[string]string `json:"locators,omitempty"`
}

type Change struct {
	BaseRevision int64      `json:"base_revision"`
	Annotation   Annotation `json:"annotation"`
}

type Conflict struct {
	Submitted Annotation `json:"submitted"`
	Current   Annotation `json:"current"`
}

type Result struct {
	Revision  int64        `json:"revision"`
	Applied   []Annotation `json:"applied,omitempty"`
	Conflicts []Conflict   `json:"conflicts,omitempty"`
}

// Export is the portable, account-scoped representation of annotation data.
// It deliberately uses explicit publication identifiers rather than the
// store's private composite map key.
type Export struct {
	Version      int                 `json:"version"`
	Publications []ExportPublication `json:"publications"`
}

type ExportPublication struct {
	LibraryID string       `json:"library_id"`
	BookID    string       `json:"book_id"`
	Revision  int64        `json:"revision"`
	Items     []Annotation `json:"items"`
}

func Valid(annotation Annotation) bool {
	if annotation.ID == "" || len(annotation.ID) > MaximumIDBytes || !utf8.ValidString(annotation.ID) || annotation.CreatedAt < 0 || annotation.UpdatedAt < 0 {
		return false
	}
	if annotation.Kind != "highlight" && annotation.Kind != "note" && annotation.Kind != "bookmark" {
		return false
	}
	if len(annotation.Quote) > MaximumQuoteBytes || len(annotation.Note) > MaximumNoteBytes || len(annotation.Color) > MaximumColorBytes || len(annotation.Chapter) > MaximumChapterBytes || len(annotation.DeviceID) > reading.MaximumDeviceIDBytes {
		return false
	}
	if !utf8.ValidString(annotation.Quote) || !utf8.ValidString(annotation.Note) || !utf8.ValidString(annotation.Chapter) || !reading.ValidAnchor(annotation.Anchor) || !reading.ValidLocators(annotation.Locators) {
		return false
	}
	return annotation.Deleted || strings.TrimSpace(annotation.Quote) != "" || strings.TrimSpace(annotation.Note) != "" || annotation.Kind == "bookmark"
}

func SameContent(left, right Annotation) bool {
	return left.ID == right.ID && left.Kind == right.Kind && left.Quote == right.Quote && left.Note == right.Note && left.Color == right.Color && left.Chapter == right.Chapter && left.CreatedAt == right.CreatedAt && left.Deleted == right.Deleted && left.DeviceID == right.DeviceID && anchorsEqual(left.Anchor, right.Anchor) && locatorsEqual(left.Locators, right.Locators)
}

func anchorsEqual(left, right *reading.Anchor) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right || (left.Version == right.Version && left.Prefix == right.Prefix && left.Exact == right.Exact && left.Suffix == right.Suffix && left.Text == right.Text && left.SpineIndex == right.SpineIndex && math.Abs(left.Progress-right.Progress) < 1e-9)
}

func locatorsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
