package web

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKoSyncStoreRejectsTrailingAndNullState(t *testing.T) {
	root := t.TempDir()
	trailing := filepath.Join(root, "trailing.json")
	if err := os.WriteFile(trailing, []byte(`{"version":1,"users":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newKoSyncStore(trailing, false); err == nil {
		t.Fatal("KoSync state with trailing JSON was accepted")
	}
	nullUser := filepath.Join(root, "null.json")
	if err := os.WriteFile(nullUser, []byte(`{"version":1,"users":{"alice":null}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newKoSyncStore(nullUser, false); err == nil {
		t.Fatal("KoSync state with a null user was accepted")
	}
}

func TestKoSyncStoreRollsBackFailedWrites(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newKoSyncStore("", false)
	if err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocked, "kosync.json")
	if err := store.ensureUser("reader", "secret"); err == nil {
		t.Fatal("user save unexpectedly succeeded")
	}
	if store.data.Users["reader"] != nil {
		t.Fatal("failed user save remained visible")
	}
	store.path = ""
	if err := store.ensureUser("reader", "secret"); err != nil {
		t.Fatal(err)
	}
	store.path = filepath.Join(blocked, "kosync.json")
	document := "0123456789abcdef0123456789abcdef"
	if _, err := store.setProgress("reader", koSyncProgress{Document: document, Progress: "/body", Timestamp: 1}); err == nil {
		t.Fatal("progress save unexpectedly succeeded")
	}
	if _, exists := store.progress("reader", document); exists {
		t.Fatal("failed progress save remained visible")
	}
}

func koSyncRequest(method, path, username, key, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("x-auth-user", username)
	request.Header.Set("x-auth-key", key)
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestKoSyncAccountProgressAndPersistence(t *testing.T) {
	database := filepath.Join(t.TempDir(), "kosync.json")
	store, err := newKoSyncStore(database, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ensureUser("reader", "secret"); err != nil {
		t.Fatal(err)
	}
	keySum := md5.Sum([]byte("secret"))
	key := hex.EncodeToString(keySum[:])

	auth := httptest.NewRecorder()
	store.serveHTTP(auth, koSyncRequest(http.MethodGet, "/users/auth", "reader", key, ""))
	if auth.Code != http.StatusOK {
		t.Fatalf("auth returned %d: %s", auth.Code, auth.Body.String())
	}

	document := "0123456789abcdef0123456789abcdef"
	body := `{"document":"` + document + `","progress":"/body/DocFragment[2]","percentage":0.42,"device":"KOReader","device_id":"device-1"}`
	put := httptest.NewRecorder()
	store.serveHTTP(put, koSyncRequest(http.MethodPut, "/syncs/progress", "reader", key, body))
	if put.Code != http.StatusOK {
		t.Fatalf("put returned %d: %s", put.Code, put.Body.String())
	}

	reloaded, err := newKoSyncStore(database, false)
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	reloaded.serveHTTP(get, koSyncRequest(http.MethodGet, "/syncs/progress/"+document, "reader", key, ""))
	var progress koSyncProgress
	if err := json.Unmarshal(get.Body.Bytes(), &progress); err != nil {
		t.Fatal(err)
	}
	if get.Code != http.StatusOK || progress.Percentage != 0.42 || progress.Progress != "/body/DocFragment[2]" || progress.Timestamp == 0 {
		t.Fatalf("unexpected progress: %d %+v", get.Code, progress)
	}

	wrong := httptest.NewRecorder()
	reloaded.serveHTTP(wrong, koSyncRequest(http.MethodGet, "/users/auth", "reader", "wrong", ""))
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key returned %d", wrong.Code)
	}
}

func TestKoSyncRegistrationPolicyAndUnknownProgress(t *testing.T) {
	store, err := newKoSyncStore(filepath.Join(t.TempDir(), "kosync.json"), false)
	if err != nil {
		t.Fatal(err)
	}
	registration := httptest.NewRecorder()
	store.serveHTTP(registration, koSyncRequest(http.MethodPost, "/users/create", "", "", `{"username":"new","password":"key"}`))
	if registration.Code != http.StatusPaymentRequired {
		t.Fatalf("registration returned %d", registration.Code)
	}
	if err := store.ensureUser("reader", "secret"); err != nil {
		t.Fatal(err)
	}
	keySum := md5.Sum([]byte("secret"))
	unknown := httptest.NewRecorder()
	store.serveHTTP(unknown, koSyncRequest(http.MethodGet, "/syncs/progress/0123456789abcdef0123456789abcdef", "reader", hex.EncodeToString(keySum[:]), ""))
	if unknown.Code != http.StatusOK || strings.TrimSpace(unknown.Body.String()) != "{}" {
		t.Fatalf("unknown progress returned %d: %s", unknown.Code, unknown.Body.String())
	}
}

func TestKoSyncPartialMD5MatchesKOReaderSampling(t *testing.T) {
	data := make([]byte, 6_000_000)
	for index := range data {
		data[index] = byte(index*37 + 11 + index>>8) // not periodic, so a wrong offset changes the hash
	}
	// Expected value computed independently with KOReader's sampling: offsets 0, 1024<<2i.
	if got := koSyncPartialMD5(bytes.NewReader(data)); got != "20d18c291f21dcefabdd77e220629893" {
		t.Fatalf("partial MD5 = %s", got)
	}
}

func TestKoSyncFilenameMD5(t *testing.T) {
	if got := koSyncFilenameMD5("book.epub"); got != "03053ffc045564439ff7f2cabb3b58c5" {
		t.Fatalf("filename MD5 = %s", got)
	}
}

func TestExternalKoSyncProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/syncs/progress/0123456789abcdef0123456789abcdef" || r.Header.Get("x-auth-user") != "reader" {
			t.Fatalf("unexpected request: %s user=%q", r.URL.Path, r.Header.Get("x-auth-user"))
		}
		sum := md5.Sum([]byte("secret"))
		if r.Header.Get("x-auth-key") != hex.EncodeToString(sum[:]) {
			t.Fatal("external KoSync key was not derived correctly")
		}
		koSyncJSON(w, http.StatusOK, koSyncProgress{Document: "0123456789abcdef0123456789abcdef", Progress: "/body/DocFragment[2]", Percentage: 0.67})
	}))
	defer server.Close()

	remote := newKoSyncRemote(server.URL, "reader", "secret")
	progress, exists, err := remote.progress(context.Background(), "0123456789abcdef0123456789abcdef")
	if err != nil || !exists || progress.Percentage != 0.67 {
		t.Fatalf("unexpected external progress: exists=%v progress=%+v err=%v", exists, progress, err)
	}
}
