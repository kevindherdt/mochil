package web

import (
	"crypto/sha256"
	"time"

	"mochil/internal/accounts"
)

// Basic-auth clients (OPDS readers, e-readers, video players) repeat their
// credentials on every request, covers included. A bcrypt comparison costs
// ~60 ms, so one OPDS page with 50 covers spent seconds of CPU re-checking the
// same correct password. A short-lived memo keeps that work to once.
const (
	// Basic-auth clients (OPDS readers, e-readers, video players) repeat their
	// credentials on every request, covers included. A bcrypt comparison costs
	// ~60 ms, so one OPDS page with 50 covers spent seconds of CPU re-checking
	// the same correct password. A short-lived memo keeps that work to once.
	basicCredentialLifetime = 5 * time.Minute
	maxBasicCredentials     = 128
)

// basicCredential is the resolved identity for one username/password digest.
// The digest is the map key; the password itself is never stored.
type basicCredential struct {
	identity accounts.Identity
	expires  time.Time
}

// basicIdentity answers a Basic-auth challenge from the memo when it can, and
// falls back to the (bcrypt) authenticator only on a miss or an expired entry.
func (a *sessionAuth) basicIdentity(username, password string) (accounts.Identity, bool) {
	key := basicCredentialKey(username, password)
	now := a.now()
	a.mu.Lock()
	if cached, ok := a.credentials[key]; ok {
		if now.Before(cached.expires) {
			a.mu.Unlock()
			// A memo entry answers faster than a bcrypt comparison, so it must not
			// outlive the account it names: a disabled or deleted account stops
			// downloading the moment it changes, even inside the memo window.
			return a.resolve(cached.identity)
		}
		delete(a.credentials, key)
	}
	a.mu.Unlock()
	// bcrypt runs for tens of milliseconds. Anything that changes an account in
	// that window clears the memo, and this check refuses to put the result back.
	generation := a.credentialGeneration.Load()
	identity, valid := a.authenticateCredentials(username, password)
	if !valid {
		// A rejected guess is never remembered: only an identity the
		// authenticator accepted enters the map.
		return accounts.Identity{}, false
	}
	if a.credentialGeneration.Load() != generation {
		return accounts.Identity{}, false
	}
	a.rememberBasic(key, identity, now)
	return identity, true
}

func (a *sessionAuth) rememberBasic(key [32]byte, identity accounts.Identity, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.credentials == nil {
		a.credentials = make(map[[32]byte]basicCredential)
	}
	if len(a.credentials) >= maxBasicCredentials {
		var oldestKey [32]byte
		var oldest time.Time
		for candidate, cached := range a.credentials {
			if oldest.IsZero() || cached.expires.Before(oldest) {
				oldestKey, oldest = candidate, cached.expires
			}
		}
		delete(a.credentials, oldestKey)
	}
	a.credentials[key] = basicCredential{identity: identity, expires: now.Add(basicCredentialLifetime)}
}

// forgetCredentials discards every memoized Basic-auth result. Account
// creation, password reset, disabling and deletion call it so a cached
// check cannot outlive the credentials it was made from.
func (a *sessionAuth) forgetCredentials() {
	a.credentialGeneration.Add(1)
	a.mu.Lock()
	a.credentials = nil
	a.mu.Unlock()
}

// basicCredentialKey hashes the presented credentials so the map key is a
// digest: the password is never stored, and two requests that send the same
// credentials land on the same entry.
func basicCredentialKey(username, password string) [32]byte {
	return sha256.Sum256([]byte(username + "\x00" + password))
}
