package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// A Basic Auth client always starts without credentials and answers the challenge; those starts are not wrong guesses.
func TestChallengesWithoutCredentialsDoNotLockEveryoneOut(t *testing.T) {
	handler := newSessionAuth("reader", "secret").wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), func(*http.Request) bool { return false })
	for attempt := 0; attempt < 30; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/opds", nil)
		request.RemoteAddr = "192.0.2.10:4000"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("challenge %d = %d", attempt+1, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/opds", nil)
	request.RemoteAddr = "192.0.2.10:4001"
	request.SetBasicAuth("reader", "secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("the right password after many challenges = %d", response.Code)
	}
}
