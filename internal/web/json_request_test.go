package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRequestAcceptsOneBoundedValue(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		maximum         int64
		disallowUnknown bool
		wantError       bool
	}{
		{name: "valid", body: `{"name":"Bobo"}`, maximum: 32},
		{name: "trailing whitespace", body: "{\"name\":\"Bobo\"}\n\t", maximum: 32},
		{name: "second value", body: `{"name":"Bobo"} {}`, maximum: 32, wantError: true},
		{name: "oversized", body: `{"name":"` + strings.Repeat("x", 64) + `"}`, maximum: 32, wantError: true},
		{name: "unknown allowed", body: `{"name":"Bobo","extra":true}`, maximum: 64},
		{name: "unknown refused", body: `{"name":"Bobo","extra":true}`, maximum: 64, disallowUnknown: true, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			var value struct {
				Name string `json:"name"`
			}
			err := decodeJSONRequest(response, request, test.maximum, &value, test.disallowUnknown)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && value.Name != "Bobo" {
				t.Fatalf("decoded name = %q", value.Name)
			}
		})
	}
}
