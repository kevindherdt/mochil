package annotations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSharedKOReaderReadiumAnnotationContract(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate interoperability contract test")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "contracts", "annotation_interop.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Version     int `json:"version"`
		Annotations []struct {
			Origin              string     `json:"origin"`
			Annotation          Annotation `json:"annotation"`
			ExpectedSpineIndex  int        `json:"expected_spine_index"`
			ExpectedKOReaderCSS string     `json:"expected_koreader_css_selector"`
		} `json:"annotations"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Version != 1 || len(contract.Annotations) != 2 {
		t.Fatalf("unexpected contract version or fixture count: %+v", contract)
	}
	for _, fixture := range contract.Annotations {
		annotation := fixture.Annotation
		if !Valid(annotation) {
			t.Fatalf("%s fixture is not a valid server annotation", fixture.Origin)
		}
		if annotation.Anchor == nil || annotation.Anchor.SpineIndex != fixture.ExpectedSpineIndex || annotation.Anchor.Exact != annotation.Quote {
			t.Fatalf("%s fixture lost its shared text anchor", fixture.Origin)
		}
		if !strings.HasPrefix(annotation.ID, fixture.Origin+":") {
			t.Fatalf("%s fixture has an unstable cross-reader identity", fixture.Origin)
		}
		if fixture.Origin == "koreader" && fixture.ExpectedKOReaderCSS == "" {
			t.Fatal("KOReader fixture does not define its equivalent Readium selector")
		}
		encoded, err := json.Marshal(annotation)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip Annotation
		if err := json.Unmarshal(encoded, &roundTrip); err != nil || !SameContent(annotation, roundTrip) {
			t.Fatalf("%s fixture did not round trip: %v", fixture.Origin, err)
		}
	}
}
