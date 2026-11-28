package web

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"

	"mochil/internal/accounts"
	"mochil/internal/acquisition"
	"mochil/internal/annotations"
	"strings"
	"testing"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/reading"
	"mochil/internal/wiki"
)

func TestKoReaderDeviceTokensSupportMultipleDevicesAndPersistence(t *testing.T) {
	path := t.TempDir() + "/devices.json"
	store, err := newKoReaderDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	var tokens []string
	for range 4 {
		token, issueErr := store.issue("reader")
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		tokens = append(tokens, token)
	}
	for index, token := range tokens {
		if account, ok := store.authenticate(token); !ok || account != "reader" {
			t.Fatalf("device %d token did not authenticate", index+1)
		}
	}
	reloaded, err := newKoReaderDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for index, token := range tokens {
		if account, ok := reloaded.authenticate(token); !ok || account != "reader" {
			t.Fatalf("device %d token did not survive reload", index+1)
		}
	}
}

func TestKoReaderDeviceStoreMigratesLegacyPairing(t *testing.T) {
	path := t.TempDir() + "/devices.json"
	token := "legacy-device-token"
	hash := sha256.Sum256([]byte(token))
	legacy, err := json.Marshal(map[string]string{"reader": base64.RawStdEncoding.EncodeToString(hash[:])})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newKoReaderDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if account, ok := store.authenticate(token); !ok || account != "reader" {
		t.Fatal("legacy device token was not migrated")
	}
}

func TestBuildKoReaderPluginIsPreconfigured(t *testing.T) {
	data, err := buildKoReaderPlugin("https://books.example", "secret-device-token")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, file := range zr.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			t.Fatal(err)
		}
		files[file.Name] = string(content)
	}
	core := files["mochil.koplugin/mochil_core.lua"]
	if core == "" || !strings.Contains(files["mochil.koplugin/main.lua"], "mochil_core.lua") {
		t.Fatal("the loader or the core is missing from the package")
	}
	config := files["mochil.koplugin/mochil_config.lua"]
	if !strings.Contains(config, "https://books.example") || !strings.Contains(config, "secret-device-token") {
		t.Fatal("generated plugin was not configured")
	}
	if !strings.Contains(core, "findAllText") {
		t.Fatal("text-anchor restore is missing")
	}
	if !strings.Contains(core, "anchor.spine_index") || !strings.Contains(core, "util.splitToChars") {
		t.Fatal("UTF-8-safe same-spine anchor matching is missing")
	}
	if !strings.Contains(core, "text_anchor=self:textAnchor") || !strings.Contains(core, "getTextFromXPointers") {
		t.Fatal("bidirectional text-anchor capture is missing")
	}
	if !strings.Contains(core, "props.display_title") || !strings.Contains(core, "props.identifiers") {
		t.Fatal("durable publication metadata is missing")
	}
	if !strings.Contains(core, "same_native or same_fraction") || !strings.Contains(core, `Event:new("GotoXPointer", native)`) {
		t.Fatal("self-push suppression or exact native restore is missing")
	}
	if !strings.Contains(core, "missing >= 2") || !strings.Contains(core, "<= 0.002") {
		t.Fatal("KOReader highlight deletion guard or equivalent-position tolerance is missing")
	}
	if !strings.Contains(core, "mochil_sync_policy") || !strings.Contains(core, "onPageUpdate") || !strings.Contains(core, "onCloseDocument") {
		t.Fatal("automatic push policies are missing")
	}
	if !strings.Contains(core, `resource or "progress"`) || !strings.Contains(core, `"annotations"`) {
		t.Fatal("annotation endpoint support is missing")
	}
	if !strings.Contains(core, "onAnnotationsModified") || !strings.Contains(core, "mochil_annotation_snapshot") {
		t.Fatal("debounced revision-aware highlight synchronization is missing")
	}
	if !strings.Contains(core, "annotationAnchor") || !strings.Contains(core, "pos0=item.pos0") {
		t.Fatal("KOReader highlight anchors and native locators are missing")
	}
	if !strings.Contains(core, "applyRemoteAnnotations") || !strings.Contains(core, "self.ui.annotation:addItem(item)") || !strings.Contains(core, "item.mochil_id") {
		t.Fatal("Android-to-KOReader highlight import or stable cross-device identity is missing")
	}
	if !strings.Contains(core, "mochil_original_id") || !strings.Contains(core, "item.pos0 == pos0") {
		t.Fatal("cross-device highlight deduplication and safe unlinking are missing")
	}
	if !strings.Contains(core, "pcall(self.request") || !strings.Contains(core, "willRerunWhenConnected") {
		t.Fatal("connection test is not guarded")
	}
	if !strings.Contains(core, `sorting_hint="tools"`) {
		t.Fatal("KOReader Tools menu placement is missing")
	}
	if !strings.Contains(core, "pcall(http.request") || !strings.Contains(core, "socketutil:reset_timeout") {
		t.Fatal("KOReader networking is not crash-guarded")
	}
}

func TestPublicationIdentityUsesStrongMetadata(t *testing.T) {
	index := 2.0
	book := catalog.Book{Title: "Rebuild World: Volume 01 Part 2", Authors: []string{"Nahuse"}, Series: "Rebuild World", SeriesIndex: &index, Identifier: "urn:isbn:9781718393144"}
	if !publicationIdentityMatchesBook(publicationIdentityEvidence{Title: book.Title, Authors: "Nahuse"}, book) {
		t.Fatal("exact title and author did not match")
	}
	if !publicationIdentityMatchesBook(publicationIdentityEvidence{Identifiers: "9781718393144"}, book) {
		t.Fatal("normalized publication identifier did not match")
	}
	if publicationIdentityMatchesBook(publicationIdentityEvidence{Title: book.Title}, book) {
		t.Fatal("title-only evidence was accepted")
	}
	if publicationIdentityMatchesBook(publicationIdentityEvidence{Series: book.Series, SeriesIndex: &index, Authors: "Someone Else"}, book) {
		t.Fatal("series identity with the wrong author was accepted")
	}
}

func TestKoReaderFilenameKeyIgnoresCommonRenamePrefixes(t *testing.T) {
	want := koReaderFilenameKey("Min-Maxing My TRPG Build - Volume 01.epub")
	for _, filename := range []string{
		"Unknown - Min-Maxing My TRPG Build - Volume 01.epub",
		"1.00 - Min-Maxing My TRPG Build - Volume 01.epub",
		"1.00 - Unknown - Min-Maxing My TRPG Build - Volume 01.epub",
	} {
		if got := koReaderFilenameKey(filename); got != want {
			t.Fatalf("koReaderFilenameKey(%q) = %q, want %q", filename, got, want)
		}
	}
}

func TestKoReaderFilenameMatchAllowsMetadataPrefix(t *testing.T) {
	catalogue := "Min-Maxing My TRPG Build in Another World - Volume 01 [J-Novel Club][Premium].epub"
	if !koReaderFilenameMatches("Schuld - "+catalogue, catalogue) {
		t.Fatal("author-prefixed download did not match catalogue filename")
	}
	if !koReaderFilenameMatches("Min-Maxing My TRPG Build in Another World - Volume 01.epub", catalogue) {
		t.Fatal("OPDS title filename did not match catalogue filename with release tags")
	}
	if koReaderFilenameMatches("Volume 01.epub", catalogue) {
		t.Fatal("short ambiguous suffix matched")
	}
}

func TestKoReaderFilenameMatchNormalizesSplitVolumeNames(t *testing.T) {
	if !koReaderFilenameMatches("Rebuild World - Volume 01 Part 2.epub", "Rebuild World - Volume 01.2 [J-Novel Club][Premium].epub") {
		t.Fatal("split volume title did not match decimal catalogue title")
	}
	if !koReaderFilenameMatches("Rebuild World - Volume 01 Part 2.epub", "Rebuild World: Volume 01 Part 2.epub") {
		t.Fatal("title punctuation prevented a match")
	}
}

func TestKoReaderFilenameMatchUsesUnambiguousCalibreSeriesPosition(t *testing.T) {
	index := 2.0
	book := catalog.Book{Title: "Rebuild World: Volume 01 Part 2", Series: "Rebuild World", SeriesIndex: &index, Authors: []string{"Nahuse"}, Path: "Nahuse/book.epub"}
	if !koReaderFilenameMatchesBook("Nahuse - Rebuild World - Volume 2.epub", book) {
		t.Fatal("Calibre series-position filename did not match its catalogue book")
	}
	if !koReaderFilenameMatchesBook("Rebuild World - Volume 01 Part 2 - Nahuse.epub", book) {
		t.Fatal("Calibre title-author filename did not match its catalogue book")
	}
	fractional := 1.2
	book.SeriesIndex = &fractional
	if koReaderFilenameMatchesBook("Nahuse - Rebuild World - Volume 1.epub", book) {
		t.Fatal("fractional series position was rounded into an unsafe match")
	}
}

func TestPreferActiveKoReaderMatchDisambiguatesDuplicateShelves(t *testing.T) {
	matches := []koReaderBookMatch{
		{libraryID: "lightnovels", bookID: "seafile-copy"},
		{libraryID: "calibrelightnovels", bookID: "calibre-copy"},
	}
	active := []readingPosition{{LibraryID: "lightnovels", BookID: "seafile-copy"}}
	got := preferActiveKoReaderMatch(matches, active)
	if len(got) != 1 || got[0] != matches[0] {
		t.Fatalf("preferred matches = %#v, want Light Novels copy", got)
	}
}

func TestPreferActiveKoReaderMatchKeepsUnsafeAmbiguity(t *testing.T) {
	matches := []koReaderBookMatch{
		{libraryID: "lightnovels", bookID: "one"},
		{libraryID: "calibrelightnovels", bookID: "two"},
	}
	if got := preferActiveKoReaderMatch(matches, nil); len(got) != 2 {
		t.Fatalf("unresolved matches = %#v, want both candidates", got)
	}
}

func TestKoReaderPluginPackageIsDeterministicAndVersioned(t *testing.T) {
	first, err := buildKoReaderPlugin("https://example.test", "token")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := buildKoReaderPlugin("https://example.test", "token")
	if !bytes.Equal(first, second) {
		t.Fatal("the plugin zip must be reproducible")
	}
	archive, err := zip.NewReader(bytes.NewReader(first), int64(len(first)))
	if err != nil {
		t.Fatal(err)
	}
	if archive.File[0].Name != "mochil.koplugin/_meta.lua" {
		t.Fatalf("unexpected first entry %q", archive.File[0].Name)
	}
	rc, _ := archive.File[0].Open()
	body, _ := io.ReadAll(rc)
	if !strings.Contains(string(body), koReaderPluginVersion) {
		t.Fatal("_meta.lua must carry the plugin version")
	}
}

func updateTestServer(t *testing.T) (*multiServer, string) {
	t.Helper()
	directory := t.TempDir()
	devices, err := newKoReaderDeviceStore(filepath.Join(directory, "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := reading.Open(filepath.Join(directory, "reading.json"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := devices.issue("reader")
	if err != nil {
		t.Fatal(err)
	}
	return &multiServer{cfg: MultiConfig{PublicURL: "https://books.example"}, devices: devices, reading: store, kosync: &koSyncStore{}}, token
}

func pluginUpdateRequest(server *multiServer, method, path, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	return response
}

func TestPluginUpdateIsOfferedToPairedDevicesOnly(t *testing.T) {
	server, token := updateTestServer(t)
	for _, path := range []string{"/api/koreader/v1/plugin/manifest", "/api/koreader/v1/plugin/core.lua"} {
		if response := pluginUpdateRequest(server, http.MethodGet, path, ""); response.Code != http.StatusUnauthorized {
			t.Fatalf("%s without a token = %d", path, response.Code)
		}
		if response := pluginUpdateRequest(server, http.MethodGet, path, "not-a-device"); response.Code != http.StatusUnauthorized {
			t.Fatalf("%s with a wrong token = %d", path, response.Code)
		}
		if response := pluginUpdateRequest(server, http.MethodPost, path, token); response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s accepted a POST: %d", path, response.Code)
		}
	}
}

func TestPluginUpdateManifestDescribesTheCoreItServes(t *testing.T) {
	server, token := updateTestServer(t)
	var manifest struct {
		Version   string `json:"version"`
		SHA256    string `json:"sha256"`
		Size      int    `json:"size"`
		LoaderAPI int    `json:"loader_api"`
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/plugin/manifest", token)
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &manifest) != nil {
		t.Fatalf("manifest = %d %s", response.Code, response.Body.String())
	}
	core := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/plugin/core.lua", token)
	if core.Code != http.StatusOK {
		t.Fatalf("core = %d", core.Code)
	}
	sum := sha256.Sum256(core.Body.Bytes())
	if manifest.SHA256 != hex.EncodeToString(sum[:]) || manifest.Size != core.Body.Len() || manifest.Version != koReaderPluginVersion || manifest.LoaderAPI != koReaderLoaderAPI {
		t.Fatalf("manifest does not match the core: %+v", manifest)
	}
	if !strings.Contains(core.Body.String(), `Mochil.release = "`+koReaderPluginVersion+`"`) {
		t.Fatal("the core reports a different version than the server advertises; update would loop")
	}
}

// The plugin's own behaviour is tested under plain Lua with a stand-in for KOReader.
func TestPluginRunsItsLuaTests(t *testing.T) {
	lua, err := exec.LookPath("lua5.4")
	if err != nil {
		t.Skip("lua5.4 is not installed")
	}
	command := exec.Command(lua, "koreader_plugin/test/run.lua")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("plugin tests failed:\n%s", output)
	}
}

func TestPluginHomeIsForPairedDevicesAndNeedsNoCatalogue(t *testing.T) {
	server, token := updateTestServer(t)
	if response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/home", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("home without a token = %d", response.Code)
	}
	if response := pluginUpdateRequest(server, http.MethodPost, "/api/koreader/v1/home", token); response.Code != http.StatusNotFound {
		t.Fatalf("home accepted a POST: %d", response.Code)
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/home", token)
	var home struct {
		Account  string            `json:"account"`
		Server   string            `json:"server"`
		Continue []json.RawMessage `json:"continue"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &home) != nil {
		t.Fatalf("home = %d %s", response.Code, response.Body.String())
	}
	if home.Account != "reader" || home.Server != "https://books.example" || home.Continue == nil {
		t.Fatalf("home = %+v (continue must be a list, not null)", home)
	}
}

func putPluginProgress(t *testing.T, server *multiServer, token, document string, percentage float64, finished bool) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"progress": "/body/DocFragment[3]/p[2]", "percentage": percentage, "device": "Kobo", "device_id": "dev1", "timestamp": time.Now().Unix(), "finished": finished})
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"linked":true,"saved":true}` {
		t.Fatalf("progress = %d %s", response.Code, response.Body.String())
	}
}

func TestABookMarkedFinishedOnTheDeviceIsReadWithoutMovingThePosition(t *testing.T) {
	server, token := browseTestServer(t)
	server.readingSync = reading.NewSyncCoordinator(server.reading)
	var err error
	if server.kosync, err = newKoSyncStore("", false); err != nil {
		t.Fatal(err)
	}
	if err := server.kosync.ensureUser("reader", "secret"); err != nil {
		t.Fatal(err)
	}
	document := strings.Repeat("ab", 16)
	if _, err := server.readingSync.Link("reader", "books", "a1", document); err != nil {
		t.Fatal(err)
	}
	putPluginProgress(t, server, token, document, 0.6, false)
	if position, _ := server.reading.Get("reader", "books", "a1"); position.Finished {
		t.Fatal("reading to 60% must not finish the book")
	}
	putPluginProgress(t, server, token, document, 0.6, true)
	position, _ := server.reading.Get("reader", "books", "a1")
	if !position.Finished || position.FinishedAt == 0 {
		t.Fatalf("the device marked it finished: %+v", position)
	}
	if position.Fraction < 0.59 || position.Fraction > 0.61 {
		t.Fatalf("marking finished moved the position to %v", position.Fraction)
	}
	// The device keeps sending the flag while the book stays marked complete, so it stays read as the page changes.
	putPluginProgress(t, server, token, document, 0.62, true)
	if position, _ := server.reading.Get("reader", "books", "a1"); !position.Finished {
		t.Fatal("a push that still carries the flag must keep the book finished")
	}
	// Once the book is no longer marked complete, the usual rule applies: below 98% it is being read again.
	putPluginProgress(t, server, token, document, 0.62, false)
	if position, _ := server.reading.Get("reader", "books", "a1"); position.Finished {
		t.Fatal("without the flag, 62% is not finished")
	}
}

func TestKoSyncMirrorUserFollowsTheAccountName(t *testing.T) {
	store, err := accounts.Open(filepath.Join(t.TempDir(), "accounts.json"), "owner", "a long enough password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create("dimadh", "another long password", accounts.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	sync, err := newKoSyncStore("", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := sync.ensureUser("dimadh", "secret"); err != nil {
		t.Fatal(err)
	}
	server := &multiServer{kosync: sync}
	server.accountsStoreField.Store(store)
	if got := server.koSyncUserFor(second.ID); got != "dimadh" {
		t.Fatalf("an account's mirror user is its user name, got %q", got)
	}
	owner, _ := store.IdentityByUsername("owner")
	if got := server.koSyncUserFor(owner.ID); got != "" {
		t.Fatalf("an account without a sync user must be skipped, got %q", got)
	}
	// A configured KoSync username belongs to the configured owner. It used to win
	// for every account, which handed each reader the owner's KoSync identity: one
	// account could read another's progress and write over it.
	server.cfg = MultiConfig{Username: "owner", KoSyncUsername: "shared"}
	if got := server.koSyncUserFor(second.ID); got != "dimadh" {
		t.Fatalf("another account keeps its own sync user, got %q", got)
	}
	if got := server.koSyncUserFor(owner.ID); got != "shared" {
		t.Fatalf("the configured owner uses the configured sync user, got %q", got)
	}
}

func TestPluginUnmatchedProgressDoesNotClaimSaved(t *testing.T) {
	server, token := browseTestServer(t)
	server.readingSync = reading.NewSyncCoordinator(server.reading)
	server.kosync, _ = newKoSyncStore("", false)
	document := strings.Repeat("ab", 16)
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, strings.NewReader(`{"progress":"/body/p[1]","percentage":0.4,"timestamp":123}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"linked":false,"saved":false}` {
		t.Fatalf("unmatched progress = %d %s", response.Code, response.Body.String())
	}
}

// Progress is acknowledged in memory and written by a coalesced background
// writer, so an unwritable state file no longer fails the request. The failure
// is reported by the store (Flush/LastError, logged at shutdown) and must still
// never leak the state path or the token into the response.
func TestPluginProgressIsAcknowledgedWhenTheReadingStoreCannotSave(t *testing.T) {
	server, token := browseTestServer(t)
	path := filepath.Join(t.TempDir(), "private-reading.json")
	store, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server.reading = store
	server.readingSync = reading.NewSyncCoordinator(store)
	server.kosync, _ = newKoSyncStore("", false)
	document := strings.Repeat("cd", 16)
	if _, err := server.readingSync.Link("reader", "books", "a1", document); err != nil {
		t.Fatal(err)
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, strings.NewReader(`{"progress":"/body/p[1]","percentage":0.4,"timestamp":123}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private-reading") || strings.Contains(response.Body.String(), token) {
		t.Fatalf("acknowledged record = %d %s", response.Code, response.Body.String())
	}
	if position, _ := store.Get("reader", "books", "a1"); position.Fraction != 0.4 {
		t.Fatalf("the acknowledged position was dropped: %+v", position)
	}
	if err := store.Flush(); err == nil {
		t.Fatal("the unreadable state path was not reported by the store")
	}
	if store.LastError() == nil {
		t.Fatal("the failed write is not visible through LastError")
	}
}

func TestPluginProgressFinishIsAcknowledgedWhenTheReadingStoreCannotSave(t *testing.T) {
	server, token := browseTestServer(t)
	path := filepath.Join(t.TempDir(), "private-reading.json")
	store, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server.reading = store
	server.readingSync = reading.NewSyncCoordinator(store)
	server.kosync, _ = newKoSyncStore("", false)
	document := strings.Repeat("ef", 16)
	if _, err := server.readingSync.Link("reader", "books", "a1", document); err != nil {
		t.Fatal(err)
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	var sabotageErr error
	store.Observe(func(_ string, _ reading.Event, _ reading.Position) {
		if sabotageErr = os.Remove(path); sabotageErr == nil {
			sabotageErr = os.Mkdir(path, 0700)
		}
	})
	body := fmt.Sprintf(`{"progress":"/body/p[1]","percentage":0.4,"timestamp":%d,"finished":true}`, time.Now().Unix())
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if sabotageErr != nil {
		t.Fatal(sabotageErr)
	}
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private-reading") || strings.Contains(response.Body.String(), token) {
		t.Fatalf("acknowledged finish = %d %s", response.Code, response.Body.String())
	}
	position, _ := store.Get("reader", "books", "a1")
	if !position.Finished || position.Fraction != 0.4 {
		t.Fatalf("unexpected result: %+v", position)
	}
	if err := store.Flush(); err == nil {
		t.Fatal("the unreadable state path was not reported by the store")
	}
}

func TestPluginProgressIsAcknowledgedWhenTheLinkStoreCannotSave(t *testing.T) {
	server, token := browseTestServer(t)
	path := filepath.Join(t.TempDir(), "private-reading.json")
	store, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server.reading = store
	server.readingSync = reading.NewSyncCoordinator(store)
	server.kosync, _ = newKoSyncStore("", false)
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	document := strings.Repeat("aa", 16)
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, strings.NewReader(`{"progress":"/body/p[1]","percentage":0.4,"timestamp":123,"filename":"Neuromancer.epub"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private-reading") || strings.Contains(response.Body.String(), token) {
		t.Fatalf("acknowledged link = %d %s", response.Code, response.Body.String())
	}
	if _, found := store.FindKoSyncDocument("reader", document); !found {
		t.Fatal("the acknowledged document link was dropped")
	}
	if err := store.Flush(); err == nil {
		t.Fatal("the unreadable state path was not reported by the store")
	}
}

func TestPluginProgressIsAcknowledgedWhenTheDigestLinkCannotSave(t *testing.T) {
	server, token := browseTestServer(t)
	path := filepath.Join(t.TempDir(), "private-reading.json")
	store, err := reading.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	server.reading = store
	server.readingSync = reading.NewSyncCoordinator(store)
	server.kosync, _ = newKoSyncStore("", false)
	if _, err := store.Put("reader", reading.Position{LibraryID: "books", BookID: "a1", Fraction: 0.3}, 0); err != nil {
		t.Fatal(err)
	}
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}
	_, book, ok := server.findBook("books/a1")
	if !ok {
		t.Fatal("book missing")
	}
	server.publications = acquisition.NewPublicationCache(t.TempDir(), 1024, 0, nil)
	content := []byte("cached publication")
	if err := os.WriteFile(server.publications.Path("books", book), content, 0600); err != nil {
		t.Fatal(err)
	}
	document := koSyncPartialMD5(bytes.NewReader(content))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, "/api/koreader/v1/progress/"+document, strings.NewReader(`{"progress":"/body/p[1]","percentage":0.4,"timestamp":123}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "private-reading") || strings.Contains(response.Body.String(), token) {
		t.Fatalf("acknowledged digest link = %d %s", response.Code, response.Body.String())
	}
	if err := store.Flush(); err == nil {
		t.Fatal("the unreadable state path was not reported by the store")
	}
}

func TestKoReaderPluginCanExportAllAnnotations(t *testing.T) {
	server, token := browseTestServer(t)
	store, err := annotations.Open(filepath.Join(t.TempDir(), "annotations.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.annotations = store
	if _, err := store.Apply("reader", "books", "a1", []annotations.Change{
		{Annotation: annotations.Annotation{ID: "mark", Kind: "highlight", Quote: "a quote", CreatedAt: 1, UpdatedAt: 1}},
	}); err != nil {
		t.Fatal(err)
	}
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/annotations/export", token)
	if response.Code != http.StatusOK {
		t.Fatalf("export = %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "a quote") {
		t.Fatalf("export should contain the reader's highlight, got %s", response.Body.String())
	}
}

func TestKoReaderPluginRejectsExportWithoutAnnotationSupport(t *testing.T) {
	server, token := browseTestServer(t)
	response := pluginUpdateRequest(server, http.MethodGet, "/api/koreader/v1/annotations/export", token)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("export without a store = %d %s, want 501", response.Code, response.Body.String())
	}
}

func TestKoReaderPluginCanExportALinkedBooksHighlightsToNotes(t *testing.T) {
	server, token := browseTestServer(t)
	annotationStore, err := annotations.Open(filepath.Join(t.TempDir(), "annotations.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.annotations = annotationStore
	wikiStore, err := wiki.Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.wiki = wikiStore
	server.readingSync = reading.NewSyncCoordinator(server.reading)
	document := strings.Repeat("ab", 16)
	if _, err := server.readingSync.Link("reader", "books", "a1", document); err != nil {
		t.Fatal(err)
	}
	if _, err := annotationStore.Apply("reader", "books", "a1", []annotations.Change{
		{Annotation: annotations.Annotation{ID: "mark", Kind: "highlight", Quote: "a quote", CreatedAt: 1, UpdatedAt: 1}},
	}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/koreader/v1/annotations/"+document+"/notes", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("notes export = %d %s", response.Code, response.Body.String())
	}
}

func TestKoReaderPluginRefusesNotesExportForAnUnlinkedDocument(t *testing.T) {
	server, token := browseTestServer(t)
	annotationStore, err := annotations.Open(filepath.Join(t.TempDir(), "annotations.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.annotations = annotationStore
	wikiStore, err := wiki.Open(filepath.Join(t.TempDir(), "wiki.json"))
	if err != nil {
		t.Fatal(err)
	}
	server.wiki = wikiStore
	server.readingSync = reading.NewSyncCoordinator(server.reading)
	document := strings.Repeat("cd", 16)
	request := httptest.NewRequest(http.MethodPost, "/api/koreader/v1/annotations/"+document+"/notes", strings.NewReader("{}"))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.koReaderPlugin(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unlinked notes export = %d %s, want 404", response.Code, response.Body.String())
	}
}
