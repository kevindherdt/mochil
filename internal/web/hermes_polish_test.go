package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"mochil/internal/appconfig"
	"mochil/internal/document"
	"mochil/internal/vaultwriter"
)

func polishServer(t *testing.T, note string, reply string, features map[string]bool) (*multiServer, string, *string, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "Linux", "keys.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	store := document.NewVaultStore("vault", root, "")
	if _, err := store.Scan(); err != nil {
		t.Fatal(err)
	}
	writer, err := vaultwriter.New(root)
	if err != nil {
		t.Fatal(err)
	}
	var sent string
	hermes := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		payload, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}}}})
		_, _ = w.Write(payload)
	}))
	t.Cleanup(hermes.Close)
	m := &multiServer{
		cfg:       MultiConfig{Hermes: appconfig.Hermes{URL: hermes.URL, Key: "k", HelperURL: hermes.URL, HelperModel: "m", Features: features}},
		libraries: []Library{{ID: "vault", Name: "Vault", Kind: "vault", Documents: store, VaultWriter: writer}},
	}
	return m, store.Items()[0].ID, &sent, path
}

func polishCall(m *multiServer, method, page string) (int, map[string]any) {
	w := httptest.NewRecorder()
	target := "/api/hermes/polish"
	var body io.Reader
	if method == http.MethodGet {
		target += "?library=vault&page=" + page
	} else {
		body = strings.NewReader(`{"library":"vault","page":"` + page + `"}`)
	}
	m.apiHermes(w, httptest.NewRequest(method, target, body))
	var state map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &state)
	return w.Code, state
}

// polishAndWait starts the job and waits for it to leave "running", the way the page does.
func polishAndWait(t *testing.T, m *multiServer, page string) (int, map[string]any) {
	t.Helper()
	code, state := polishCall(m, http.MethodPost, page)
	if code != http.StatusAccepted {
		return code, state
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if code, state = polishCall(m, http.MethodGet, page); state["status"] != "running" {
			return code, state
		}
	}
	t.Fatal("polish did not finish")
	return 0, nil
}

func TestHermesPolishProposesWithoutSavingAndKeepsFrontMatter(t *testing.T) {
	note := "---\ntags: [linux]\n---\nkeys\nsee [[Server]]\nrestart: sudo systemctl restart x\n"
	reply := "# Keys\n\n- See [[Server]]\n- Restart: `sudo systemctl restart x`"
	m, page, sent, path := polishServer(t, note, reply, nil)
	code, state := polishAndWait(t, m, page)
	if code != http.StatusOK || state["status"] != "ready" {
		t.Fatalf("polish = %d %v", code, state)
	}
	got, _ := state["result"].(map[string]any)
	proposal, _ := got["proposal"].(string)
	if got["original"] != note || !strings.HasPrefix(proposal, "---\ntags: [linux]\n---\n# Keys") || !strings.HasSuffix(proposal, "`\n") {
		t.Fatalf("proposal = %q", proposal)
	}
	if lost, _ := got["lost"].([]any); len(lost) != 0 || got["same"] != false || got["writable"] != true || got["revision"] != vaultwriter.Revision([]byte(note)) {
		t.Fatalf("result = %+v", got)
	}
	if strings.Contains(*sent, "tags: [linux]") || !strings.Contains(*sent, "see [[Server]]") {
		t.Fatalf("front matter must stay out of the request: %s", *sent)
	}
	if onDisk, _ := os.ReadFile(path); string(onDisk) != note {
		t.Fatalf("the note must not be changed by a proposal: %q", onDisk)
	}
}

func TestHermesPolishReportsWhatTheProposalLost(t *testing.T) {
	note := "keys\nsee [[Server]]\ntoken eyJhbGciOiJIUzI1NiJ9.abcdefghijklmnopqrstuvwxyz0123\n"
	reply := "# Keys\n\nSee the server page.\nToken: eyJhbGciOiJ..."
	m, page, _, _ := polishServer(t, note, reply, nil)
	_, state := polishAndWait(t, m, page)
	raw, _ := json.Marshal(state)
	if !strings.Contains(string(raw), "link: [[Serv") || !strings.Contains(string(raw), "key or token:") {
		t.Fatalf("lost items expected: %s", raw)
	}
	if strings.Contains(string(raw), `"lost":["key or token: eyJhbGciOiJIUzI1NiJ9.abcdefghijklmnopqrstuvwxyz0123`) {
		t.Fatalf("a secret was echoed in the lost list: %s", raw)
	}
}

func TestHermesPolishRefusesAnAnswerThatIsNotTheNoteAndHonoursTheSwitch(t *testing.T) {
	note := strings.Repeat("a line of the note\n", 20)
	m, page, _, _ := polishServer(t, note, "Sorry, I can't help with that.", nil)
	if _, state := polishAndWait(t, m, page); state["status"] != "failed" || state["result"] != nil {
		t.Fatalf("a refusal must not be proposed: %v", state)
	}
	off, page, _, _ := polishServer(t, note, note, map[string]bool{"polish": false})
	if code, _ := polishCall(off, http.MethodPost, page); code != http.StatusForbidden {
		t.Fatalf("switched off = %d", code)
	}
}

func TestPolishRunsOnAfterTheRequestAndCanBePickedUpOrStopped(t *testing.T) {
	note := strings.Repeat("a line of the note\n", 20)
	m, page, _, _ := polishServer(t, note, note, nil)
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":`+strconv.Quote(note)+`}}]}`)
	}))
	defer slow.Close()
	m.cfg.Hermes.HelperURL = slow.URL

	code, state := polishCall(m, http.MethodPost, page)
	if code != http.StatusAccepted || state["status"] != "running" {
		t.Fatalf("start must answer at once: %d %v", code, state)
	}
	if code, state = polishCall(m, http.MethodPost, page); code != http.StatusOK || state["status"] != "running" {
		t.Fatalf("starting the same note again picks the work back up: %d %v", code, state)
	}
	if _, state = polishCall(m, http.MethodGet, page); state["status"] != "running" {
		t.Fatalf("status = %v", state)
	}
	// Stop really stops it, and frees Hermes for the next request.
	polishCall(m, http.MethodDelete, page)
	if _, state = polishCall(m, http.MethodGet, page); state["status"] != "idle" {
		t.Fatalf("after Stop = %v", state)
	}
	for deadline := time.Now().Add(2 * time.Second); !m.tryHermesTurn(); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("Hermes was not freed after Stop")
		}
	}
	// While Hermes is busy with something else, a new request is told so.
	if code, _ = polishCall(m, http.MethodPost, page); code != http.StatusTooManyRequests {
		t.Fatalf("busy = %d", code)
	}
	m.endHermesTurn()
	close(release)
}
