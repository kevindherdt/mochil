package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"mochil/internal/catalog"
)

func TestAndroidPublicationPageRequiresDeviceTokenAndServesPageBytes(t *testing.T) {
	backend := streamingBackend{book: catalog.Book{
		ID: "comic", Path: "Comic.cbz", Title: "Comic", Format: "cbz",
		MediaType: "application/vnd.comicbook+zip", PageCount: 3, PageMediaType: "image/jpeg",
	}}
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := multiServer{cfg: MultiConfig{Username: "reader"}, androidDevices: devices,
		libraries: []Library{{ID: "manga", Kind: "manga", Store: backend, PageStreaming: true}}}
	token, deviceID, err := devices.issue(m.readingAccount(nil), "Fixture phone")
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, credential string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if credential != "" {
			r.Header.Set("Authorization", "Bearer "+credential)
		}
		response := httptest.NewRecorder()
		m.androidAPI(response, r)
		return response
	}
	path := "/api/android/v1/publications/manga/comic/pages/1"

	if response := request(path, ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status=%d", response.Code)
	}
	if response := request(path, "not-a-real-token"); response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status=%d", response.Code)
	}

	response := request(path, token)
	if response.Code != http.StatusOK || response.Body.String() != "page" || response.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("authenticated page fetch status=%d type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}

	if outOfRange := request("/api/android/v1/publications/manga/comic/pages/99", token); outOfRange.Code != http.StatusBadRequest {
		t.Fatalf("out of range page status=%d", outOfRange.Code)
	}
	if missingBook := request("/api/android/v1/publications/manga/missing/pages/0", token); missingBook.Code != http.StatusNotFound {
		t.Fatalf("missing book status=%d", missingBook.Code)
	}

	if err := devices.revoke(m.readingAccount(nil), deviceID); err != nil {
		t.Fatal(err)
	}
	if revoked := request(path, token); revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked device status=%d", revoked.Code)
	}
}

func TestAndroidPublicationPageIsNotAdvertisedWhenPageStreamingIsDisabled(t *testing.T) {
	backend := streamingBackend{book: catalog.Book{
		ID: "comic", Path: "Comic.cbz", Title: "Comic", Format: "cbz",
		MediaType: "application/vnd.comicbook+zip", PageCount: 3, PageMediaType: "image/jpeg",
	}}
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := multiServer{cfg: MultiConfig{Username: "reader"}, androidDevices: devices,
		libraries: []Library{{ID: "manga", Kind: "manga", Store: backend, PageStreaming: false}}}
	token, _, err := devices.issue(m.readingAccount(nil), "Fixture phone")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/android/v1/publications/manga/comic/pages/1", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	m.androidAPI(response, r)
	if response.Code != http.StatusNotFound {
		t.Fatalf("page streaming disabled status=%d", response.Code)
	}
}
