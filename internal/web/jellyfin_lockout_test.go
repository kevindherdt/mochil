package web

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJellyfinSignInLocksOutPasswordGuessing(t *testing.T) {
	handler, err := NewMulti(MultiConfig{Username: "reader", Password: "secret"}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer handler.(interface{ Close() }).Close()
	attempt := func(password string) int {
		request := httptest.NewRequest(http.MethodPost, "/Users/AuthenticateByName", bytes.NewBufferString(`{"Username":"reader","Pw":"`+password+`"}`))
		request.RemoteAddr = "203.0.113.9:5555"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	for i := 0; i < jellyfinFailureLimit; i++ {
		if code := attempt("wrong"); code != http.StatusUnauthorized {
			t.Fatalf("guess %d: status %d", i, code)
		}
	}
	if code := attempt("secret"); code != http.StatusTooManyRequests {
		t.Fatalf("even the right password must wait once locked out, got %d", code)
	}
}
