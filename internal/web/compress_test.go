package web

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, contentType string, header http.Header, body string) *httptest.ResponseRecorder {
	t.Helper()
	handler := Compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.Header().Set("Etag", `"abc"`)
		_, _ = io.WriteString(w, body)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	for key, values := range header {
		request.Header[key] = values
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestCompressGzipsTextAndJSON(t *testing.T) {
	body := strings.Repeat(`{"title":"book"}`, 200)
	for _, contentType := range []string{"application/json", "text/css; charset=utf-8", "text/javascript; charset=utf-8", "application/atom+xml;profile=opds-catalog"} {
		recorder := serve(t, contentType, nil, body)
		if recorder.Header().Get("Content-Encoding") != "gzip" || recorder.Header().Get("Etag") != `W/"abc"` {
			t.Fatalf("%s: headers %v", contentType, recorder.Header())
		}
		reader, err := gzip.NewReader(recorder.Body)
		if err != nil {
			t.Fatal(err)
		}
		decoded, _ := io.ReadAll(reader)
		if string(decoded) != body || recorder.Body.Len() >= len(body) {
			t.Fatalf("%s: round trip failed", contentType)
		}
	}
}

func TestCompressLeavesMediaRangesAndEncodedAlone(t *testing.T) {
	for name, test := range map[string]struct {
		contentType string
		header      http.Header
	}{
		"image":  {"image/jpeg", nil},
		"epub":   {"application/epub+zip", nil},
		"audio":  {"audio/mpeg", nil},
		"events": {"text/event-stream", nil},
		"range":  {"application/json", http.Header{"Range": {"bytes=0-9"}}},
	} {
		recorder := serve(t, test.contentType, test.header, "payload")
		if recorder.Header().Get("Content-Encoding") != "" || recorder.Body.String() != "payload" {
			t.Fatalf("%s was altered: %v", name, recorder.Header())
		}
	}
}

func TestCompressSkipsClientsWithoutGzip(t *testing.T) {
	handler := Compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "{}")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Header().Get("Content-Encoding") != "" || recorder.Body.String() != "{}" {
		t.Fatal("response must stay plain")
	}
}

func TestCompressRestoresStrongETagForConditionalRequests(t *testing.T) {
	var seen string
	handler := Compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusNotModified)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("If-None-Match", `W/"abc"`)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if seen != `"abc"` || recorder.Code != http.StatusNotModified || recorder.Header().Get("Content-Encoding") != "" {
		t.Fatalf("seen %q code %d headers %v", seen, recorder.Code, recorder.Header())
	}
}
