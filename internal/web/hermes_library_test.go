package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/appconfig"
	"mochil/internal/audio"
)

func TestHermesPlayWordsOnlyForLibraryPlayRequests(t *testing.T) {
	cases := []struct {
		message string
		want    string
	}{
		{"hey lyana can u play Lilac by IU", "lilac iu"},
		{"Please put on some Daft Punk", "daft punk"},
		{"Play IU LILAC from my music library.", "iu lilac"},
		{"play lilac by iu locally from the downloaded files", "lilac iu"},
		{"just local playback here in the app", ""},
		{"play radio maria on the living room speaker", ""},
		{"play studio brussel in the bedroom", ""},
		{"what is the weather", ""},
		{"play some music", ""},
	}
	for _, c := range cases {
		if got := strings.Join(hermesPlayWords(c.message, "Lyana"), " "); got != c.want {
			t.Errorf("%q -> %q, want %q", c.message, got, c.want)
		}
	}
}

func TestMatchAudioItemsNeedsEveryWordAndPutsTitleMatchesFirst(t *testing.T) {
	items := []audio.Item{
		{ID: "1", Title: "Palette", Artist: "IU", Album: "Lilac fan edition"},
		{ID: "2", Title: "Lilac", Artist: "IU"},
		{ID: "3", Title: "Lilac", Artist: "Someone Else"},
	}
	got := matchAudioItems(items, []string{"lilac", "iu"}, 5)
	if len(got) != 2 || got[0].ID != "2" || got[1].ID != "1" {
		t.Fatalf("matches: %#v", got)
	}
	long := []audio.Item{{ID: "a", Title: "IU - IU 5th Album 'LILAC' - 01-06 Troll"}, {ID: "b", Title: "LILAC", Artist: "IU"}}
	if got := matchAudioItems(long, []string{"lilac", "iu"}, 5); got[0].ID != "b" {
		t.Fatalf("the closer title must come first: %#v", got)
	}
	if len(matchAudioItems(items, []string{"lilac", "nothing"}, 5)) != 0 || len(matchAudioItems(items, nil, 5)) != 0 {
		t.Fatal("a missing word or no words must match nothing")
	}
	if len(matchAudioItems(items, []string{"lilac"}, 1)) != 1 {
		t.Fatal("limit not applied")
	}
}

func TestHermesChatShowsPlayCardsAndTellsTheAssistant(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"IU/Lilac.mp3", "IU/Palette.mp3"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := audio.NewStore("music", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	var upstream string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		upstream = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Found it\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer hermes.Close()
	m := &multiServer{cfg: MultiConfig{Username: "Kev", Hermes: appconfig.Hermes{URL: hermes.URL + "/", Key: "secret", Name: "Lyana"}}, libraries: []Library{{ID: "music", Name: "Music", Kind: "music", Audio: store}}}

	w := httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"can u play lilac"}`)))
	body := w.Body.String()
	if !strings.Contains(body, "event: hermes.cards") || !strings.Contains(body, `"title":"Lilac"`) || !strings.Contains(body, "/audio/music/") {
		t.Fatalf("no card: %q", body)
	}
	if strings.Contains(body, `"title":"Palette"`) {
		t.Fatalf("an unrelated track was offered: %q", body)
	}
	if upstream != "" || !strings.Contains(body, "Tap Play") || !strings.Contains(body, "[DONE]") {
		t.Fatalf("a play request must be answered by the server without asking the model: upstream=%q body=%q", upstream, body)
	}

	w = httptest.NewRecorder()
	m.apiHermes(w, httptest.NewRequest(http.MethodPost, "/api/hermes/chat", strings.NewReader(`{"session":"c1234567890abcdef","message":"play lilac on the living room speaker"}`)))
	if strings.Contains(w.Body.String(), "hermes.cards") || strings.Contains(w.Body.String(), "Tap Play") || upstream == "" {
		t.Fatalf("a speaker request must not search the library: %q", w.Body.String())
	}
}

func TestMatchAudioItemsForgivesATypoOnlyWhenNothingMatchesExactly(t *testing.T) {
	items := []audio.Item{{ID: "1", Title: "LILAC", Artist: "IU"}, {ID: "2", Title: "Flu", Artist: "IU"}}
	if got := matchAudioItems(items, []string{"lilak", "iu"}, 5); len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("typo: %#v", got)
	}
	if got := matchAudioItems(items, []string{"iu"}, 5); len(got) != 2 {
		t.Fatalf("exact matches must not be widened: %#v", got)
	}
	if got := matchAudioItems(items, []string{"xyz"}, 5); len(got) != 0 {
		t.Fatalf("short unrelated words must not match: %#v", got)
	}
}

func TestPlayExchangesComeBackInTheirPlaceWithTheirCards(t *testing.T) {
	store := newHermesPlayStore(filepath.Join(t.TempDir(), "plays.json"))
	card := hermesCard{ID: "7", Library: "music", Title: "Lilac"}
	store.add("c1", hermesPlay{At: 2, Before: 2, User: "play lilac", Reply: "Found it", Cards: []hermesCard{card}})
	own := []hermesMessage{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "hello"}, {Role: "user", Text: "thanks"}}
	got := store.merge("c1", own)
	if len(got) != 5 || got[2].Text != "play lilac" || got[3].Cards[0].Title != "Lilac" || got[4].Text != "thanks" {
		t.Fatalf("merged: %#v", got)
	}
	again := newHermesPlayStore(store.path)
	if len(again.merge("c1", own)) != 5 {
		t.Fatal("not saved to disk")
	}
	store.forget("c1")
	if len(store.merge("c1", own)) != 3 {
		t.Fatal("not forgotten")
	}
}
