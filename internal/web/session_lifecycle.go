package web

import (
	"time"

	"mochil/internal/accounts"
)

// Session issuance, revocation and persistence live here so the request flow in
// session_auth.go stays readable: everything below decides which cookies exist,
// not which one a request arrived with.

func (a *sessionAuth) createSession(identity accounts.Identity, remember bool) (string, bool) {
	token, ok := randomToken(32)
	if !ok {
		return "", false
	}
	now := a.now()
	a.mu.Lock()
	a.pruneLocked(now)
	if len(a.sessions) >= maxSessions {
		var oldestKey [32]byte
		var oldest time.Time
		for key, session := range a.sessions {
			if oldest.IsZero() || session.seen.Before(oldest) {
				oldestKey, oldest = key, session.seen
			}
		}
		delete(a.sessions, oldestKey)
	}
	a.sessions[sessionKey(token)] = webSession{created: now, seen: now, identity: identity, remember: remember}
	a.mu.Unlock()
	if remember {
		a.persist()
	}
	return token, true
}

// revokeAccount signs an account out everywhere in the browser.
func (a *sessionAuth) revokeAccount(account string) error {
	// Ending someone's sign-ins also invalidates any password check still in
	// flight for them: see credentialGeneration.
	a.credentialGeneration.Add(1)
	a.mu.Lock()
	for key, session := range a.sessions {
		if session.identity.ID == account {
			delete(a.sessions, key)
		}
	}
	a.mu.Unlock()
	return a.persist()
}

// persist writes the remembered sessions to disk, when a store is configured.
func (a *sessionAuth) persist() error {
	if a.store == nil {
		return nil
	}
	a.persistMu.Lock()
	defer a.persistMu.Unlock()
	a.mu.Lock()
	snapshot := make(map[[32]byte]webSession, len(a.sessions))
	for key, session := range a.sessions {
		if session.remember {
			snapshot[key] = session
		}
	}
	covered := make([][32]byte, 0, len(a.pendingRevocations))
	for key := range a.pendingRevocations {
		covered = append(covered, key)
	}
	a.lastSaved = a.now()
	a.mu.Unlock()
	if err := a.store.save(snapshot); err != nil {
		return err
	}
	a.mu.Lock()
	a.persistedSessions = snapshot
	for _, key := range covered {
		delete(a.pendingRevocations, key)
	}
	a.mu.Unlock()
	return nil
}
