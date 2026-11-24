package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// A compressed body gets a weak ETag (the middleware rewrites the handler's strong one), so a client echoes `W/"x"`
// back as its precondition. The handler compares strong values, so the prefix has to be stripped on the way in.
//
// This is not hypothetical: with `If-Match` missing from that strip, every audio-playlist edit answered 412
// "changed on another device" to a client that had changed nothing — rename, reorder, remove and delete alike.
func TestAPreconditionQuotedFromACompressedResponseReachesTheHandlerUnprefixed(t *testing.T) {
	for _, test := range []struct {
		name   string
		header string
		value  string
		want   string
	}{
		{"if-match weak", "If-Match", `W/"abc123"`, `"abc123"`},
		{"if-none-match weak", "If-None-Match", `W/"abc123"`, `"abc123"`},
		{"if-match already strong", "If-Match", `"abc123"`, `"abc123"`},
		{"if-match any", "If-Match", "*", "*"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var seen string
			handler := Compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = r.Header.Get(test.header)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("ETag", `"abc123"`)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			request := httptest.NewRequest(http.MethodPut, "/api/audio-playlists/Night%20drive", nil)
			request.Header.Set("Accept-Encoding", "gzip")
			request.Header.Set(test.header, test.value)
			handler.ServeHTTP(httptest.NewRecorder(), request)

			if seen != test.want {
				t.Fatalf("%s reaching the handler = %q, want %q", test.header, seen, test.want)
			}
		})
	}
}

// The other half of the same contract: the response really is weak, which is *why* the request side has to cope.
func TestACompressedJSONResponseAdvertisesItsETagAsWeak(t *testing.T) {
	handler := Compress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"abc123"`)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/audio-playlists/Night%20drive", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("ETag"); got != `W/"abc123"` {
		t.Fatalf("ETag = %q, want the weak form %q", got, `W/"abc123"`)
	}
}
