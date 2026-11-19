package reading

import (
	"math"
	"strings"
	"testing"
)

func TestValidationBounds(t *testing.T) {
	p := Position{LibraryID: "books", BookID: "one", Fraction: .5, WebAnchor: &Anchor{Exact: "words", SpineIndex: 2, Progress: .4}}
	if !ValidUpdate(p) {
		t.Fatal("rejected valid position")
	}
	p.Fraction = math.NaN()
	if ValidUpdate(p) {
		t.Fatal("accepted NaN")
	}
	p.Fraction = .5
	p.WebAnchor = &Anchor{Exact: strings.Repeat("x", MaximumAnchorPartBytes+1), SpineIndex: 1}
	if ValidUpdate(p) {
		t.Fatal("accepted oversized anchor")
	}
	p.WebAnchor = nil
	p.CFI = strings.Repeat("x", MaximumCFIBytes+1)
	if ValidUpdate(p) {
		t.Fatal("accepted oversized CFI")
	}
	p.CFI = ""
	p.DeviceName = strings.Repeat("x", MaximumDeviceNameBytes+1)
	if ValidUpdate(p) {
		t.Fatal("accepted oversized device name")
	}
}

func TestKoReaderLocatorValidation(t *testing.T) {
	valid := []string{"", "/body/DocFragment[4]/body/div[1]/p[3]/text().25", "/body/DocFragment[1].0/body/section/p"}
	for _, locator := range valid {
		if !ValidKoReaderLocator(locator) {
			t.Fatalf("rejected valid KOReader locator %q", locator)
		}
	}
	invalid := []string{"/body/DocFragment[0]/body/p", "/body/DocFragment[1]/body/p:nth-child(1)", "/body/DocFragment[1]/body/text().0/p[1]", strings.Repeat("x", MaximumKoReaderLocatorBytes+1)}
	for _, locator := range invalid {
		if ValidKoReaderLocator(locator) {
			t.Fatalf("accepted invalid KOReader locator %q", locator)
		}
	}
}
