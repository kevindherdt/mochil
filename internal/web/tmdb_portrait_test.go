package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"mochil/internal/enrichment"
)

func TestTMDBPortraitBrowserSession(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: metadataRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(strings.NewReader("image"))}, nil
	})}
	m := &multiServer{metadataArtwork: enrichment.NewArtworkCache(t.TempDir(), 10000, client)}
	handler := newSessionAuth("reader", "secret").wrap(m, func(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, androidPrefix) })
	endpoint := "/api/tmdb-portrait?path=%2Fabc.jpg"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, endpoint, nil))
	if response.Code != http.StatusSeeOther || calls != 0 {
		t.Fatalf("unauthenticated status=%d upstream=%d", response.Code, calls)
	}
	login := httptest.NewRecorder()
	handler.ServeHTTP(login, loginPOST(t, handler, url.Values{"username": {"reader"}, "password": {"secret"}}))
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie")
	}
	request := httptest.NewRequest(http.MethodGet, endpoint, nil)
	request.AddCookie(cookies[0])
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "image" || calls != 1 {
		t.Fatalf("authenticated status=%d body=%q upstream=%d", response.Code, response.Body.String(), calls)
	}
}

func TestTMDBPortraitAndroidProxy(t *testing.T) {
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := &http.Client{Transport: metadataRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://image.tmdb.org/t/p/w342/abc.jpg" {
			t.Errorf("unexpected upstream %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(strings.NewReader("image"))}, nil
	})}
	m := multiServer{cfg: MultiConfig{Username: "reader"}, androidDevices: devices, metadataArtwork: enrichment.NewArtworkCache(t.TempDir(), 10000, client)}
	token, _, err := devices.issue(m.readingAccount(nil), "phone")
	if err != nil {
		t.Fatal(err)
	}
	request := func(path, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		return w
	}
	endpoint := "/api/android/v1/tmdb-portrait?path=%2Fabc.jpg"
	if got := request(endpoint, ""); got.Code != 401 {
		t.Fatalf("unauthenticated=%d", got.Code)
	}
	if got := request(endpoint, token); got.Code != 200 || got.Body.String() != "image" {
		t.Fatalf("status=%d body=%q", got.Code, got.Body.String())
	} else if got.Header().Get("Cache-Control") != "private, no-store" {
		t.Errorf("android Cache-Control=%q", got.Header().Get("Cache-Control"))
	}
	if got := request("/api/tmdb-portrait?path=%2Fabc.jpg", ""); got.Code != 200 || got.Header().Get("Cache-Control") != "private, max-age=86400" {
		t.Errorf("browser status=%d Cache-Control=%q", got.Code, got.Header().Get("Cache-Control"))
	}
	m.cfg.Modules = map[string]bool{"android": true, "video": false}
	for _, p := range []string{endpoint, "/api/tmdb-portrait?path=%2Fabc.jpg"} {
		if got := request(p, token); got.Code != http.StatusNotFound {
			t.Errorf("disabled video path=%s status=%d", p, got.Code)
		}
	}
	m.cfg.Modules = nil
	for _, path := range []string{"%2F..%2Fevil.jpg", "https%3A%2F%2Fevil.test%2Fx.jpg", "%2Fa.jpg%3Fx%3D1", "%2Ffoo%252Fbar.jpg"} {
		if got := request("/api/android/v1/tmdb-portrait?path="+path, token); got.Code != 400 {
			t.Errorf("path=%s status=%d", path, got.Code)
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d", calls)
	}
}
