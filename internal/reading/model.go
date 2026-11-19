package reading

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
)

const (
	// PositionEquivalenceFraction treats renderer rounding within 0.2% of a
	// publication as the same effective reading position across clients.
	PositionEquivalenceFraction = 0.002
	MaximumCFIBytes             = 8192
	MaximumDeviceIDBytes        = 128
	MaximumDeviceNameBytes      = 128
	MaximumAnchorTextBytes      = 1024
	MaximumAnchorPartBytes      = 2048
	MaximumAnchorBytes          = 6144
	MaximumLocatorTypes         = 8
	MaximumLocatorKeyBytes      = 48
	MaximumLocatorBytes         = 16384
	MaximumLocatorsBytes        = 32768
	MaximumKoReaderLocatorBytes = 8192
)

var koReaderLocatorPattern = regexp.MustCompile(`^/body/DocFragment\[[1-9][0-9]{0,8}\](?:\.[0-9]+)?(?:/body(?:/[A-Za-z][A-Za-z0-9_-]*(?:\[[1-9][0-9]*\])?)*(?:/text\(\)(?:\[[1-9][0-9]*\])?(?:\.[0-9]+)?)?)?$`)

var ErrConflict = errors.New("reading position changed in another browser")
var ErrInvalid = errors.New("invalid reading position")

type Position struct {
	LibraryID       string            `json:"library_id"`
	BookID          string            `json:"book_id"`
	CFI             string            `json:"cfi,omitempty"`
	Fraction        float64           `json:"fraction"`
	UpdatedAt       int64             `json:"updated_at"`
	Revision        int64             `json:"revision"`
	DeviceID        string            `json:"device_id,omitempty"`
	DeviceName      string            `json:"device_name,omitempty"`
	Finished        bool              `json:"finished,omitempty"`
	FinishedAt      int64             `json:"finished_at,omitempty"`
	Dismissed       bool              `json:"dismissed,omitempty"`
	Status          string            `json:"status,omitempty"` // on_hold or dropped; empty otherwise
	KoSyncSeenAt    int64             `json:"kosync_seen_at,omitempty"`
	KoSyncDocument  string            `json:"kosync_document,omitempty"`
	KoSyncDocuments []string          `json:"kosync_documents,omitempty"`
	KoReaderLocator string            `json:"koreader_locator,omitempty"`
	WebAnchor       *Anchor           `json:"web_anchor,omitempty"`
	Locators        map[string]string `json:"locators,omitempty"`
}
type Anchor struct {
	Version    int     `json:"version,omitempty"`
	Prefix     string  `json:"prefix,omitempty"`
	Exact      string  `json:"exact,omitempty"`
	Suffix     string  `json:"suffix,omitempty"`
	Text       string  `json:"text,omitempty"`
	SpineIndex int     `json:"spine_index"`
	Progress   float64 `json:"progress,omitempty"`
}

func ValidFraction(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }
func EquivalentFraction(left, right float64) bool {
	return ValidFraction(left) && ValidFraction(right) && math.Abs(left-right) <= PositionEquivalenceFraction
}

// EquivalentPosition keeps renderer tolerance for reflowable publications but
// treats fixed-layout pages as exact. On long PDFs or comics, adjacent pages can
// be less than PositionEquivalenceFraction apart and must never be acknowledged
// as the same position.
func EquivalentPosition(left, right Position) bool {
	leftPage, leftHasFixed, leftValid := fixedLayoutPage(left.Locators)
	rightPage, rightHasFixed, rightValid := fixedLayoutPage(right.Locators)
	if leftHasFixed || rightHasFixed {
		return leftHasFixed && rightHasFixed && leftValid && rightValid && leftPage == rightPage
	}
	return EquivalentFraction(left.Fraction, right.Fraction) || sameReflowableLocation(left, right)
}

func sameReflowableLocation(left, right Position) bool {
	if left.CFI != "" && left.CFI == right.CFI {
		return true
	}
	if left.KoReaderLocator != "" && left.KoReaderLocator == right.KoReaderLocator {
		return true
	}
	return equivalentAnchor(left.WebAnchor, right.WebAnchor)
}

func equivalentAnchor(left, right *Anchor) bool {
	if left == nil || right == nil || left.SpineIndex != right.SpineIndex {
		return false
	}
	leftExact, rightExact := left.Exact, right.Exact
	if leftExact == "" {
		leftExact = left.Text
	}
	if rightExact == "" {
		rightExact = right.Text
	}
	if leftExact == "" || leftExact != rightExact {
		return false
	}
	// Context is required because a short repeated sentence is not a unique
	// reading location. Progress is deliberately excluded: renderer layout is
	// the source of the harmless fraction disagreement handled here.
	if left.Prefix == "" && left.Suffix == "" || right.Prefix == "" && right.Suffix == "" {
		return false
	}
	return left.Prefix == right.Prefix && left.Suffix == right.Suffix
}

func fixedLayoutPage(locators map[string]string) (page int, present bool, valid bool) {
	for _, kind := range []string{"comic", "pdf"} {
		encoded, ok := locators[kind]
		if !ok {
			continue
		}
		var locator struct {
			Page *int `json:"page"`
		}
		if err := json.Unmarshal([]byte(encoded), &locator); err != nil || locator.Page == nil || *locator.Page < 0 {
			return 0, true, false
		}
		return *locator.Page, true, true
	}
	return 0, false, false
}
func ValidAnchor(a *Anchor) bool {
	if a == nil {
		return true
	}
	if a.SpineIndex < 0 || !ValidFraction(a.Progress) {
		return false
	}
	return len(a.Text) <= MaximumAnchorTextBytes && len(a.Prefix) <= MaximumAnchorPartBytes && len(a.Exact) <= MaximumAnchorPartBytes && len(a.Suffix) <= MaximumAnchorPartBytes && len(a.Text)+len(a.Prefix)+len(a.Exact)+len(a.Suffix) <= MaximumAnchorBytes
}
func ValidUpdate(p Position) bool {
	return p.LibraryID != "" && p.BookID != "" && len(p.CFI) <= MaximumCFIBytes && len(p.DeviceID) <= MaximumDeviceIDBytes && len(p.DeviceName) <= MaximumDeviceNameBytes && ValidFraction(p.Fraction) && ValidAnchor(p.WebAnchor) && ValidLocators(p.Locators) && ValidKoReaderLocator(p.KoReaderLocator)
}

func ValidKoReaderLocator(locator string) bool {
	return locator == "" || len(locator) <= MaximumKoReaderLocatorBytes && koReaderLocatorPattern.MatchString(locator)
}

func ValidLocators(locators map[string]string) bool {
	if len(locators) > MaximumLocatorTypes {
		return false
	}
	total := 0
	for kind, locator := range locators {
		if kind == "" || len(kind) > MaximumLocatorKeyBytes || len(locator) > MaximumLocatorBytes {
			return false
		}
		total += len(kind) + len(locator)
	}
	return total <= MaximumLocatorsBytes
}
