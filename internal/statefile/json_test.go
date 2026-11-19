package statefile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeJSONRejectsOversizeTrailingAndNonRegularInput(t *testing.T) {
	directory := t.TempDir()
	for name, body := range map[string]string{"oversize.json": `{"value":"too large"}`, "trailing.json": `{"value":"ok"} {}`} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var value map[string]string
	if err := DecodeJSON(filepath.Join(directory, "oversize.json"), 4, &value); err == nil {
		t.Fatal("expected oversized state to fail")
	}
	if err := DecodeJSON(filepath.Join(directory, "trailing.json"), 1<<10, &value); err == nil {
		t.Fatal("expected trailing JSON to fail")
	}
	if err := DecodeJSON(directory, 1<<10, &value); err == nil {
		t.Fatal("expected directory input to fail")
	}
}

func BenchmarkDecodeJSONCatalogue100k(b *testing.B) {
	type entry struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Size  int64  `json:"size"`
	}
	payload := struct {
		Version int     `json:"version"`
		Entries []entry `json:"entries"`
	}{Version: 1, Entries: make([]entry, 100_000)}
	for index := range payload.Entries {
		payload.Entries[index] = entry{ID: fmt.Sprintf("item-%06d", index), Title: fmt.Sprintf("Catalogue item %06d", index), Size: int64(index)}
	}
	path := filepath.Join(b.TempDir(), "catalogue.json")
	data, err := json.Marshal(payload)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(data))/(1<<20), "MiB/file")
	b.ResetTimer()
	for range b.N {
		var decoded struct {
			Version int     `json:"version"`
			Entries []entry `json:"entries"`
		}
		if err := DecodeJSON(path, MaximumCatalogueBytes, &decoded); err != nil || len(decoded.Entries) != 100_000 {
			b.Fatalf("decode = %d entries, %v", len(decoded.Entries), err)
		}
	}
}

func TestDecodeJSONAcceptsOneDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"value":"ok"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var value map[string]string
	if err := DecodeJSON(path, 1<<10, &value); err != nil || value["value"] != "ok" {
		t.Fatalf("decoded state = %#v, %v", value, err)
	}
}

func TestEncodeJSONEnforcesLimit(t *testing.T) {
	var output bytes.Buffer
	if err := EncodeJSON(&output, 8, map[string]string{"value": "too large"}, false); err == nil {
		t.Fatal("expected bounded encode to fail")
	}
	output.Reset()
	if err := EncodeJSON(&output, 1<<10, map[string]string{"value": "ok"}, false); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil || decoded["value"] != "ok" {
		t.Fatalf("encoded state = %q, %v", output.Bytes(), err)
	}
}

func TestEncodeJSONBudgetCountsEscapingIndentAndTerminatingNewline(t *testing.T) {
	value := map[string]string{"text": "日本語 <tag>\n\"quoted\""}
	for _, indent := range []bool{false, true} {
		var expected bytes.Buffer
		encoder := json.NewEncoder(&expected)
		if indent {
			encoder.SetIndent("", "  ")
		}
		if err := encoder.Encode(value); err != nil {
			t.Fatal(err)
		}
		var actual bytes.Buffer
		if err := EncodeJSON(&actual, int64(expected.Len()), value, indent); err != nil {
			t.Fatalf("exact budget rejected (indent=%v): %v", indent, err)
		}
		if !bytes.Equal(actual.Bytes(), expected.Bytes()) {
			t.Fatal("serialization format changed")
		}
		actual.Reset()
		if err := EncodeJSON(&actual, int64(expected.Len()-1), value, indent); err == nil {
			t.Fatalf("one-byte overflow accepted (indent=%v)", indent)
		}
		if actual.Len() != 0 {
			t.Fatal("oversized document partially written")
		}
	}
}
