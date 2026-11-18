package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func koSyncAuthRequest(address, user, key string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/users/auth", nil)
	request.RemoteAddr = address + ":5000"
	request.Header.Set("x-auth-user", user)
	request.Header.Set("x-auth-key", key)
	return request
}

func TestKoSyncGuessesAreLimitedPerAddressAndUser(t *testing.T) {
	store, err := newKoSyncStore("", false)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Now()
	store.now = func() time.Time { return current }
	if err := store.ensureUser("alice", "secret"); err != nil {
		t.Fatal(err)
	}
	if err := store.ensureUser("bob", "other"); err != nil {
		t.Fatal(err)
	}
	right := "5ebe2294ecd0e0f08eab7690d2a6ee69" // md5("secret")
	for range koSyncFailureLimit {
		if _, ok := store.authenticate(koSyncAuthRequest("10.0.0.1", "alice", "wrong")); ok {
			t.Fatal("a wrong key was accepted")
		}
	}
	if _, ok := store.authenticate(koSyncAuthRequest("10.0.0.1", "alice", right)); ok {
		t.Fatal("the limit did not stop further guesses at the same user")
	}
	if _, ok := store.authenticate(koSyncAuthRequest("10.0.0.1", "bob", "5b2c7a2c1a2f8d5c3f6a1b7a0e9d4c3b")); ok {
		t.Fatal("a wrong key for another user was accepted")
	}
	if _, ok := store.authenticate(koSyncAuthRequest("10.0.0.2", "alice", right)); !ok {
		t.Fatal("another address was locked out by someone else's guesses")
	}
	current = current.Add(koSyncFailureWindow + time.Second)
	if _, ok := store.authenticate(koSyncAuthRequest("10.0.0.1", "alice", right)); !ok {
		t.Fatal("the lock-out did not end after its window")
	}
}
