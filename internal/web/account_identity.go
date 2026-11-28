package web

import (
	"crypto/subtle"
	"net/http"

	"mochil/internal/accounts"
)

func (m *multiServer) authenticateAccount(username, password string) (accounts.Identity, bool) {
	if m.accountStore() != nil {
		if identity, err := m.accountStore().Authenticate(username, password); err == nil {
			return identity, true
		}
	}
	// The environment credential remains a recovery/bootstrap path until the
	// persisted administrator is explicitly confirmed through account settings.
	// It only exists when a credential was actually configured: comparing two
	// empty strings succeeds, so without this guard a blank sign-in on a
	// first-run install (where nothing was configured and recovery is still on)
	// turned into the administrator.
	if m.cfg.Username == "" || m.cfg.Password == "" {
		return accounts.Identity{}, false
	}
	if m.accountStore() != nil && !m.accountStore().RecoveryEnabled() {
		return accounts.Identity{}, false
	}
	goodUser := subtle.ConstantTimeCompare([]byte(username), []byte(m.cfg.Username)) == 1
	goodPass := subtle.ConstantTimeCompare([]byte(password), []byte(m.cfg.Password)) == 1
	if !goodUser || !goodPass {
		return accounts.Identity{}, false
	}
	if m.accountStore() != nil {
		if identity, ok := m.accountStore().InitialAdministrator(); ok {
			return identity, true
		}
	}
	return accounts.Identity{ID: m.cfg.Username, Username: m.cfg.Username, Role: accounts.RoleAdministrator}, true
}

func (m *multiServer) resolveAccountIdentity(identity accounts.Identity) (accounts.Identity, bool) {
	if m.accountStore() == nil {
		return identity, identity.ID != ""
	}
	return m.accountStore().IdentityByID(identity.ID)
}

// forgetBasicCredentials drops memoized Basic-auth results after an account is
// created, disabled, password-reset or deleted: a cached check must not keep
// serving credentials that no longer exist.
func (m *multiServer) forgetBasicCredentials() {
	if m.forgetBasicAuth != nil {
		m.forgetBasicAuth()
	}
}

func (m *multiServer) readingAccount(r *http.Request) string {
	if identity, ok := identityFromRequest(r); ok {
		return identity.ID
	}
	if m.cfg.Username != "" {
		return m.cfg.Username
	}
	return "default"
}
