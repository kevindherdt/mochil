package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadRedactsAndUpdatePreservesPassword(t *testing.T) {
	stored := []byte(`{"kosync":{"username":"reader","password":"secret"},"libraries":[]}`)
	s := Service{ReadSource: func() ([]byte, error) { return stored, nil }, SaveSource: func(data []byte) error { stored = append([]byte(nil), data...); return nil }, EffectiveUser: "reader", PasswordConfigured: true}
	shown, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(shown), "secret") || !strings.Contains(string(shown), "password_configured") {
		t.Fatalf("unsafe response: %s", shown)
	}
	if _, err = s.Update([]byte(`{"kosync":{"username":"reader","password":""},"libraries":[]}`)); err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err = json.Unmarshal(stored, &saved); err != nil {
		t.Fatal(err)
	}
	sync := saved["kosync"].(map[string]any)
	if sync["password"] != "secret" {
		t.Fatalf("password was not preserved: %#v", sync)
	}
}

func TestRejectsNonObjectAndOversize(t *testing.T) {
	s := Service{ReadSource: func() ([]byte, error) { return []byte(`{}`), nil }, SaveSource: func([]byte) error { return nil }}
	if _, err := s.Update([]byte(`[]`)); err == nil {
		t.Fatal("accepted array")
	}
	if _, err := s.Update(make([]byte, MaximumBytes+1)); err == nil {
		t.Fatal("accepted oversized settings")
	}
}

func TestProviderTokensAreRedactedAndPreserved(t *testing.T) {
	stored := []byte(`{"metadata_providers":{"hardcover_token":"book-secret","tmdb_token":"video-secret"},"libraries":[]}`)
	s := Service{ReadSource: func() ([]byte, error) { return stored, nil }, SaveSource: func(data []byte) error { stored = append([]byte(nil), data...); return nil }}
	shown, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(shown), "secret") || !strings.Contains(string(shown), "hardcover_token_configured") || !strings.Contains(string(shown), "tmdb_token_configured") {
		t.Fatalf("unsafe provider response: %s", shown)
	}
	if _, err = s.Update([]byte(`{"metadata_providers":{"hardcover_token":"","tmdb_token":""},"libraries":[]}`)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stored), "book-secret") || !strings.Contains(string(stored), "video-secret") {
		t.Fatalf("tokens were not preserved: %s", stored)
	}
}

func TestRetiredExternalSourcesAreNeitherExposedNorSaved(t *testing.T) {
	stored := []byte(`{"external_sources":{"retired":{"url":"https://source.example","username":"reader","password":"source-secret"}},"libraries":[]}`)
	s := Service{ReadSource: func() ([]byte, error) { return stored, nil }, SaveSource: func(data []byte) error { stored = append([]byte(nil), data...); return nil }}
	shown, err := s.Read()
	if err != nil || strings.Contains(string(shown), "external_sources") || strings.Contains(string(shown), "source-secret") {
		t.Fatalf("shown=%s err=%v", shown, err)
	}
	if _, err = s.Update([]byte(`{"external_sources":{"retired":{"url":"https://source.example","password":"new-secret"}},"libraries":[]}`)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "external_sources") || strings.Contains(string(stored), "secret") {
		t.Fatalf("stored=%s", stored)
	}
}

func TestHermesKeyIsRedactedAndKeptWhenLeftBlank(t *testing.T) {
	source := `{"hermes":{"url":"http://h:8642","key":"secret","name":"Mochi"}}`
	var saved []byte
	svc := Service{ReadSource: func() ([]byte, error) { return []byte(source), nil }, SaveSource: func(b []byte) error { saved = b; return nil }}
	out, err := svc.Read()
	if err != nil || strings.Contains(string(out), "secret") || !strings.Contains(string(out), `"key_configured":true`) {
		t.Fatalf("read: %v %s", err, out)
	}
	if _, err := svc.Update([]byte(`{"hermes":{"url":"http://h:9","key":"","key_configured":true,"name":"Pip"}}`)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), `"key": "secret"`) || !strings.Contains(string(saved), "Pip") || strings.Contains(string(saved), "key_configured") {
		t.Fatalf("kept: %s", saved)
	}
	if _, err := svc.Update([]byte(`{"hermes":{"key":"new"}}`)); err != nil || !strings.Contains(string(saved), `"key": "new"`) {
		t.Fatalf("replace: %v %s", err, saved)
	}
}

func TestTheNarrationTokenNeverLeavesTheServer(t *testing.T) {
	config := []byte(`{"narration":{"token":"hf_secret_value","engine":"pocket"},"libraries":[]}`)
	service := Service{ReadSource: func() ([]byte, error) { return config, nil }, SaveSource: func(data []byte) error { config = data; return nil }}
	document, err := service.Read()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(document), "hf_secret_value") || strings.Contains(string(document), `"token"`) {
		t.Fatalf("the narration token reached the browser: %s", document)
	}
	if !strings.Contains(string(document), `"narration_token_configured":true`) {
		t.Fatalf("the page cannot say a token is saved: %s", document)
	}
	// Saving everything else must not wipe the token the page never saw.
	edited := strings.Replace(string(document), `"engine":"pocket"`, `"engine":"pocket","threads":4`, 1)
	if _, err := service.Update([]byte(edited)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "hf_secret_value") {
		t.Fatalf("the saved token was lost by a settings save: %s", config)
	}
	if strings.Contains(string(config), "narration_token_configured") {
		t.Fatalf("a display-only field was written into the config: %s", config)
	}
}
