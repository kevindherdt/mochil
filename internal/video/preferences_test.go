package video

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreferenceStoreRejectsTrailingState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := os.WriteFile(path, []byte(`{} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenPreferenceStore(path, LanguagePreferences{}); err == nil {
		t.Fatal("video preferences with trailing JSON were accepted")
	}
}

func TestPreferenceStorePersistsPerShowAndLibrary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	store, err := OpenPreferenceStore(path, LanguagePreferences{AudioLanguage: "Japanese", SubtitleLanguage: "English"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("anime", "A Show", LanguagePreferences{AudioLanguage: "Korean", SubtitleLanguage: "off"}); err != nil {
		t.Fatal(err)
	}
	effective, override := store.Effective("anime", "A Show")
	if effective.AudioLanguage != "korean" || effective.SubtitleLanguage != "off" || override.AudioLanguage != "korean" {
		t.Fatalf("unexpected preferences: %#v %#v", effective, override)
	}
	other, _ := store.Effective("tv", "A Show")
	if other.AudioLanguage != "japanese" || other.SubtitleLanguage != "english" {
		t.Fatalf("library override leaked: %#v", other)
	}
	reopened, err := OpenPreferenceStore(path, LanguagePreferences{})
	if err != nil {
		t.Fatal(err)
	}
	persisted, _ := reopened.Effective("anime", "A Show")
	if persisted.AudioLanguage != "korean" || persisted.SubtitleLanguage != "off" {
		t.Fatalf("preferences not persisted: %#v", persisted)
	}
}

func TestPreferenceStoreRollsBackFailedSave(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenPreferenceStore("", LanguagePreferences{AudioLanguage: "English"})
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocked, "preferences.json")
	if err := store.Set("tv", "A Show", LanguagePreferences{AudioLanguage: "Japanese"}); err == nil {
		t.Fatal("save unexpectedly succeeded")
	}
	effective, override := store.Effective("tv", "A Show")
	if effective.AudioLanguage != "english" || override.AudioLanguage != "" {
		t.Fatalf("failed save remained visible: effective=%#v override=%#v", effective, override)
	}
}
