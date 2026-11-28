package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/annotations"
	"mochil/internal/audio"
	"mochil/internal/externallibrary"
	"mochil/internal/reading"
	"mochil/internal/video"
	"mochil/internal/wiki"
)

func TestAdministratorAccountAPIKeepsPasswordsWriteOnly(t *testing.T) {
	directory := t.TempDir()
	handler, err := NewMulti(MultiConfig{Username: "owner", Password: "owner password", ReadingState: filepath.Join(directory, "reading.json")}, []Library{{ID: "books", Name: "Books", Kind: "books", Store: staticBackend{}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	loginRequest := loginPOST(t, handler, url.Values{"username": {"owner"}, "password": {"owner password"}})
	login := httptest.NewRecorder()
	handler.ServeHTTP(login, loginRequest)
	cookie := login.Result().Cookies()[0]

	create := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewBufferString(`{"username":"reader","password":"reader password","role":"user"}`))
	create.Header.Set("Content-Type", "application/json")
	create.AddCookie(cookie)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated || strings.Contains(created.Body.String(), "password") {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	list.AddCookie(cookie)
	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"username":"reader"`) || strings.Contains(listed.Body.String(), "password") || !strings.Contains(listed.Body.String(), `"registration_enabled":false`) {
		t.Fatalf("list=%d %s", listed.Code, listed.Body.String())
	}

	var response struct {
		Items []struct{ ID, Username string } `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	readerID := ""
	for _, item := range response.Items {
		if item.Username == "reader" {
			readerID = item.ID
		}
	}
	reset := httptest.NewRequest(http.MethodPatch, "/api/accounts", bytes.NewBufferString(`{"id":"`+readerID+`","action":"reset_password","password":"replacement password"}`))
	reset.Header.Set("Content-Type", "application/json")
	reset.AddCookie(cookie)
	resetResponse := httptest.NewRecorder()
	handler.ServeHTTP(resetResponse, reset)
	if resetResponse.Code != http.StatusNoContent {
		t.Fatalf("reset=%d %s", resetResponse.Code, resetResponse.Body.String())
	}
}

func TestAccountExportIsBoundedPasswordFreeAndIsolated(t *testing.T) {
	directory := t.TempDir()
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "never-export-owner")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "never-export-reader", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	readingStore, _ := reading.Open("")
	annotationStore, _ := annotations.Open("")
	audioStore, _ := audio.OpenProgressStore("")
	if _, err = readingStore.Put(reader.ID, reading.Position{LibraryID: "books", BookID: "reader-book", Fraction: .4}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = readingStore.Put("owner", reading.Position{LibraryID: "books", BookID: "owner-book", Fraction: .8}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = annotationStore.Apply(reader.ID, "books", "reader-book", []annotations.Change{{Annotation: annotations.Annotation{ID: "highlight-1", Kind: "highlight", Quote: "reader quote", CreatedAt: 1, UpdatedAt: 1}}}); err != nil {
		t.Fatal(err)
	}
	if err = audioStore.Put(reader.ID, audio.Progress{LibraryID: "music", ItemID: "track-1", Position: 12, Duration: 120}); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{reading: readingStore, annotations: annotationStore, audioProgress: audioStore}
	m.accountsStoreField.Store(accountStore)
	owner, _ := accountStore.IdentityByID("owner")
	request := httptest.NewRequest(http.MethodGet, "/api/accounts/"+url.PathEscape(reader.ID)+"/export", nil)
	request = request.WithContext(contextWithIdentity(request.Context(), owner))
	response := httptest.NewRecorder()
	m.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Disposition") != `attachment; filename="mochil-account-export.json"` || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("export=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
	body := response.Body.String()
	for _, forbidden := range []string{"never-export-owner", "never-export-reader", "owner-book", "password_hash", "device_token"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("export leaked %q: %s", forbidden, body)
		}
	}
	var exported accountDataExport
	if err = json.Unmarshal(response.Body.Bytes(), &exported); err != nil {
		t.Fatal(err)
	}
	if exported.Account.ID != reader.ID || exported.Account.Username != "reader" || len(exported.Reading.Items) != 1 || exported.Reading.Items[0].Position.BookID != "reader-book" || len(exported.AudioProgress) != 1 || !strings.Contains(string(exported.Annotations), "reader quote") {
		t.Fatalf("unexpected export: %#v annotations=%s", exported, exported.Annotations)
	}
	if exported.ExportedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("stale export time: %v", exported.ExportedAt)
	}

	denied := httptest.NewRequest(http.MethodGet, "/api/accounts/"+url.PathEscape(reader.ID)+"/export", nil)
	denied = denied.WithContext(contextWithIdentity(denied.Context(), reader))
	deniedResponse := httptest.NewRecorder()
	m.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusForbidden {
		t.Fatalf("ordinary account export status = %d", deniedResponse.Code)
	}
}

func TestDisabledAccountDeletionPurgesEveryPersonalStoreAndCredential(t *testing.T) {
	directory := t.TempDir()
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	readingStore, _ := reading.Open(filepath.Join(directory, "reading.json"))
	annotationStore, _ := annotations.Open(filepath.Join(directory, "annotations.json"))
	audioProgress, _ := audio.OpenProgressStore(filepath.Join(directory, "audio.json"))
	videoProgress, _ := video.OpenProgressStore(filepath.Join(directory, "video.json"))
	queue, _ := audio.OpenQueueStore(filepath.Join(directory, "queue.json"))
	favorites, _ := audio.OpenQueueStore(filepath.Join(directory, "favorites.json"))
	playlists, _ := audio.OpenPlaylistStore(filepath.Join(directory, "playlists.json"))
	collections, _ := newCollectionStore(filepath.Join(directory, "collections.json"))
	wikiStore, _ := wiki.Open(filepath.Join(directory, "wiki.json"))
	external, _ := externallibrary.Open(filepath.Join(directory, "external.json"))
	androidDevices, _ := newAndroidDeviceStore(filepath.Join(directory, "android.json"))
	jellyfinDevices, _ := newAndroidDeviceStore(filepath.Join(directory, "jellyfin.json"))
	koDevices, _ := newKoReaderDeviceStore(filepath.Join(directory, "koreader.json"))
	if _, err = readingStore.Put(reader.ID, reading.Position{LibraryID: "books", BookID: "book", Fraction: .3}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = annotationStore.Apply(reader.ID, "books", "book", []annotations.Change{{Annotation: annotations.Annotation{ID: "mark", Kind: "highlight", Quote: "text", CreatedAt: 1, UpdatedAt: 1}}}); err != nil {
		t.Fatal(err)
	}
	if err = audioProgress.Put(reader.ID, audio.Progress{LibraryID: "music", ItemID: "track", Position: 1, Duration: 10}); err != nil {
		t.Fatal(err)
	}
	if err = videoProgress.Put(reader.ID, video.Progress{LibraryID: "movies", ItemID: "film", Position: 2, Duration: 20}); err != nil {
		t.Fatal(err)
	}
	entry := audio.QueueEntry{LibraryID: "music", ItemID: "track"}
	if err = queue.Put(reader.ID, []audio.QueueEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err = favorites.Put(reader.ID, []audio.QueueEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err = playlists.Put(reader.ID, "Saved", []audio.QueueEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if _, err = collections.create(reader.ID, "Shelf", "books", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = wikiStore.Create(reader.ID, "Note", "private"); err != nil {
		t.Fatal(err)
	}
	title, err := external.Put(reader.ID, externallibrary.Title{ProviderID: "provider", SourceID: "source", RemoteTitleID: "remote", Name: "Manga", ExecutionLocation: externallibrary.ExecutionDevice})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = external.PutProgress(reader.ID, title.ID, externallibrary.Progress{RemoteChapterID: "chapter", Fraction: .5}, 0); err != nil {
		t.Fatal(err)
	}
	androidToken, _, _ := androidDevices.issue(reader.ID, "Phone")
	jellyfinToken, _, _ := jellyfinDevices.issue(reader.ID, "Television")
	koToken, _ := koDevices.issue(reader.ID)
	if _, err = readingStore.Put("owner", reading.Position{LibraryID: "books", BookID: "owner-book", Fraction: .7}, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = annotationStore.Apply("owner", "books", "owner-book", []annotations.Change{{Annotation: annotations.Annotation{ID: "owner-mark", Kind: "highlight", Quote: "owner text", CreatedAt: 1, UpdatedAt: 1}}}); err != nil {
		t.Fatal(err)
	}
	if err = audioProgress.Put("owner", audio.Progress{LibraryID: "music", ItemID: "owner-track", Position: 3, Duration: 10}); err != nil {
		t.Fatal(err)
	}
	if err = videoProgress.Put("owner", video.Progress{LibraryID: "movies", ItemID: "owner-film", Position: 4, Duration: 20}); err != nil {
		t.Fatal(err)
	}
	ownerEntry := audio.QueueEntry{LibraryID: "music", ItemID: "owner-track"}
	if err = queue.Put("owner", []audio.QueueEntry{ownerEntry}); err != nil {
		t.Fatal(err)
	}
	if err = favorites.Put("owner", []audio.QueueEntry{ownerEntry}); err != nil {
		t.Fatal(err)
	}
	if err = playlists.Put("owner", "Owner saved", []audio.QueueEntry{ownerEntry}); err != nil {
		t.Fatal(err)
	}
	if _, err = collections.create("owner", "Owner shelf", "books", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = wikiStore.Create("owner", "Owner note", "keep"); err != nil {
		t.Fatal(err)
	}
	ownerTitle, err := external.Put("owner", externallibrary.Title{ProviderID: "provider", SourceID: "source", RemoteTitleID: "owner-remote", Name: "Owner manga", ExecutionLocation: externallibrary.ExecutionDevice})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = external.PutProgress("owner", ownerTitle.ID, externallibrary.Progress{RemoteChapterID: "owner-chapter", Fraction: .6}, 0); err != nil {
		t.Fatal(err)
	}
	ownerAndroidToken, _, _ := androidDevices.issue("owner", "Owner phone")
	ownerJellyfinToken, _, _ := jellyfinDevices.issue("owner", "Owner television")
	ownerKoToken, _ := koDevices.issue("owner")
	if err = accountStore.SetDisabled(reader.ID, true); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{reading: readingStore, annotations: annotationStore, audioProgress: audioProgress, videoProgress: videoProgress, audioQueue: queue, audioFavorites: favorites, audioPlaylists: playlists, collections: collections, wiki: wikiStore, externalLibrary: external, androidDevices: androidDevices, jellyfinDevices: jellyfinDevices, devices: koDevices}
	m.accountsStoreField.Store(accountStore)
	owner, _ := accountStore.IdentityByID("owner")

	wrongResponse := httptest.NewRecorder()
	m.ServeHTTP(wrongResponse, accountDeleteRequest(reader.ID, "wrong", owner))
	if wrongResponse.Code != http.StatusBadRequest || len(readingStore.ExportAccount(reader.ID).Items) != 1 {
		t.Fatalf("wrong confirmation = %d", wrongResponse.Code)
	}
	response := httptest.NewRecorder()
	m.ServeHTTP(response, accountDeleteRequest(reader.ID, "reader", owner))
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete=%d %s", response.Code, response.Body.String())
	}
	if _, found := m.accountSummary(reader.ID); found {
		t.Fatal("credential record remained")
	}
	if len(readingStore.ExportAccount(reader.ID).Items) != 0 || len(audioProgress.List(reader.ID)) != 0 || len(videoProgress.List(reader.ID)) != 0 || len(queue.Get(reader.ID)) != 0 || len(favorites.Get(reader.ID)) != 0 || len(playlists.List(reader.ID)) != 0 || len(collections.list(reader.ID)) != 0 || len(wikiStore.List(reader.ID, false)) != 0 || len(external.List(reader.ID)) != 0 {
		t.Fatal("personal state remained after deletion")
	}
	annotationData, _ := annotationStore.ExportAccount(reader.ID)
	if strings.Contains(string(annotationData), "mark") {
		t.Fatal("annotations remained")
	}
	if _, _, ok := androidDevices.authenticate(androidToken); ok {
		t.Fatal("Android token remained")
	}
	if _, _, ok := jellyfinDevices.authenticate(jellyfinToken); ok {
		t.Fatal("Jellyfin token remained")
	}
	if _, ok := koDevices.authenticate(koToken); ok {
		t.Fatal("KOReader token remained")
	}
	if len(readingStore.ExportAccount("owner").Items) != 1 || len(audioProgress.List("owner")) != 1 || len(videoProgress.List("owner")) != 1 || len(queue.Get("owner")) != 1 || len(favorites.Get("owner")) != 1 || len(playlists.List("owner")) != 1 || len(collections.list("owner")) != 1 || len(wikiStore.List("owner", false)) != 1 || len(external.List("owner")) != 1 {
		t.Fatal("another account's personal state was changed")
	}
	ownerAnnotations, _ := annotationStore.ExportAccount("owner")
	if !strings.Contains(string(ownerAnnotations), "owner-mark") {
		t.Fatal("another account's annotations were changed")
	}
	if account, _, ok := androidDevices.authenticate(ownerAndroidToken); !ok || account != "owner" {
		t.Fatal("another account's Android token was revoked")
	}
	if account, _, ok := jellyfinDevices.authenticate(ownerJellyfinToken); !ok || account != "owner" {
		t.Fatal("another account's Jellyfin token was revoked")
	}
	if account, ok := koDevices.authenticate(ownerKoToken); !ok || account != "owner" {
		t.Fatal("another account's KOReader token was revoked")
	}
}

func TestInterruptedAccountDeletionRemainsDisabledAndCanBeRetried(t *testing.T) {
	directory := t.TempDir()
	accountStore, _ := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	reader, _ := accountStore.Create("reader", "reader password", accounts.RoleUser)
	readingPath := filepath.Join(directory, "reading.json")
	readingStore, _ := reading.Open(readingPath)
	_, _ = readingStore.Put(reader.ID, reading.Position{LibraryID: "books", BookID: "book", Fraction: .2}, 0)
	if err := readingStore.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := accountStore.SetDisabled(reader.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(readingPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(readingPath, 0o700); err != nil {
		t.Fatal(err)
	}
	m := &multiServer{reading: readingStore}
	m.accountsStoreField.Store(accountStore)
	owner, _ := accountStore.IdentityByID("owner")
	failed := httptest.NewRecorder()
	m.ServeHTTP(failed, accountDeleteRequest(reader.ID, "reader", owner))
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("interrupted status=%d %s", failed.Code, failed.Body.String())
	}
	if summary, found := m.accountSummary(reader.ID); !found || !summary.Disabled {
		t.Fatalf("interrupted account=%#v found=%v", summary, found)
	}
	if len(readingStore.ExportAccount(reader.ID).Items) != 1 {
		t.Fatal("failed purge was not rolled back")
	}
	if err := os.Remove(readingPath); err != nil {
		t.Fatal(err)
	}
	retried := httptest.NewRecorder()
	m.ServeHTTP(retried, accountDeleteRequest(reader.ID, "reader", owner))
	if retried.Code != http.StatusNoContent {
		t.Fatalf("retry=%d %s", retried.Code, retried.Body.String())
	}
}

func accountDeleteRequest(id, confirmation string, identity accounts.Identity) *http.Request {
	request := httptest.NewRequest(http.MethodDelete, "/api/accounts/"+url.PathEscape(id), strings.NewReader(`{"confirmation":"`+confirmation+`"}`))
	request.Header.Set("Content-Type", "application/json")
	return request.WithContext(contextWithIdentity(request.Context(), identity))
}

func TestPasswordResetEndsEveryExistingSignIn(t *testing.T) {
	directory := t.TempDir()
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	devices, _ := newAndroidDeviceStore(filepath.Join(directory, "android.json"))
	jellyfin, _ := newAndroidDeviceStore(filepath.Join(directory, "jellyfin.json"))
	koDevices, _ := newKoReaderDeviceStore(filepath.Join(directory, "koreader.json"))
	androidToken, _, _ := devices.issue(reader.ID, "Phone")
	jellyfinToken, _, _ := jellyfin.issue(reader.ID, "TV")
	dropped := ""
	m := &multiServer{androidDevices: devices, jellyfinDevices: jellyfin, devices: koDevices, dropSessions: func(id string) error { dropped = id; return nil }}
	m.accountsStoreField.Store(accountStore)
	response := httptest.NewRecorder()
	m.applyAccountAction(response, httptest.NewRequest(http.MethodPost, "/api/accounts/"+reader.ID, nil), reader.ID, "reset_password", "a brand new password")
	if response.Code != http.StatusNoContent {
		t.Fatalf("reset status = %d %s", response.Code, response.Body.String())
	}
	if dropped != reader.ID {
		t.Fatalf("browser sessions dropped for %q", dropped)
	}
	if _, _, ok := devices.authenticate(androidToken); ok {
		t.Fatal("Android token survived the password reset")
	}
	if _, _, ok := jellyfin.authenticate(jellyfinToken); ok {
		t.Fatal("Jellyfin token survived the password reset")
	}
}

func TestPasswordResetReportsSignInsThatCouldNotBeEnded(t *testing.T) {
	directory := t.TempDir()
	accountStore, err := accounts.Open(filepath.Join(directory, "accounts.json"), "owner", "owner password")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := accountStore.Create("reader", "reader password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	m := &multiServer{dropSessions: func(string) error { return errors.New("disk full") }}
	m.accountsStoreField.Store(accountStore)
	response := httptest.NewRecorder()
	m.applyAccountAction(response, httptest.NewRequest(http.MethodPost, "/api/accounts/"+reader.ID, nil), reader.ID, "reset_password", "a brand new password")
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "Reset it again") {
		t.Fatalf("a failed revocation must be visible: %d %s", response.Code, response.Body.String())
	}
	if _, err := accountStore.Authenticate("reader", "a brand new password"); err != nil {
		t.Fatal("the password change itself should have been kept")
	}
}
