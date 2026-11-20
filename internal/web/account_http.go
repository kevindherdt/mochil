package web

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"mochil/internal/accounts"
)

func (m *multiServer) accountAPI(w http.ResponseWriter, r *http.Request) {
	if m.accounts == nil {
		http.Error(w, "Persistent accounts are unavailable", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/accounts"), "/")
	if suffix == "" {
		switch r.Method {
		case http.MethodGet:
			identity, _ := identityFromRequest(r)
			writeJSON(w, http.StatusOK, map[string]any{"items": m.accounts.List(), "current_account_id": identity.ID, "registration_enabled": false, "environment_recovery_enabled": m.accounts.RecoveryEnabled()})
		case http.MethodPost:
			var input struct {
				Username string `json:"username"`
				Password string `json:"password"`
				Role     string `json:"role"`
			}
			if decodeJSONRequest(w, r, 16<<10, &input, true) != nil {
				http.Error(w, "Invalid account", http.StatusBadRequest)
				return
			}
			identity, err := m.accounts.Create(input.Username, input.Password, input.Role)
			if err != nil {
				writeAccountError(w, err)
				return
			}
			m.forgetBasicCredentials()
			writeJSON(w, http.StatusCreated, identity)
		case http.MethodPatch:
			var input struct {
				ID       string `json:"id"`
				Action   string `json:"action"`
				Password string `json:"password,omitempty"`
			}
			if decodeJSONRequest(w, r, 16<<10, &input, true) != nil {
				http.Error(w, "Invalid account action", http.StatusBadRequest)
				return
			}
			m.applyAccountAction(w, r, input.ID, input.Action, input.Password)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if encodedID, ok := strings.CutSuffix(suffix, "/export"); ok {
		id, err := url.PathUnescape(encodedID)
		if err != nil || id == "" || strings.Contains(id, "/") {
			http.Error(w, "Invalid account", http.StatusBadRequest)
			return
		}
		m.exportAccount(w, r, id)
		return
	}
	if r.Method == http.MethodDelete {
		id, err := url.PathUnescape(suffix)
		if err != nil || id == "" || strings.Contains(id, "/") {
			http.Error(w, "Invalid account", http.StatusBadRequest)
			return
		}
		var input struct {
			Confirmation string `json:"confirmation"`
		}
		if decodeJSONRequest(w, r, 16<<10, &input, true) != nil {
			http.Error(w, "Invalid account deletion", http.StatusBadRequest)
			return
		}
		m.deleteAccount(w, r, id, input.Confirmation)
		return
	}
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := url.PathUnescape(suffix)
	if err != nil || id == "" || strings.Contains(id, "/") {
		http.Error(w, "Invalid account", http.StatusBadRequest)
		return
	}
	var input struct {
		Action   string `json:"action"`
		Password string `json:"password,omitempty"`
	}
	if decodeJSONRequest(w, r, 16<<10, &input, true) != nil {
		http.Error(w, "Invalid account action", http.StatusBadRequest)
		return
	}
	m.applyAccountAction(w, r, id, input.Action, input.Password)
}

func (m *multiServer) applyAccountAction(w http.ResponseWriter, r *http.Request, id, action, password string) {
	m.accountMutationMu.Lock()
	defer m.accountMutationMu.Unlock()
	current, _ := identityFromRequest(r)
	var err error
	switch action {
	case "disable":
		if current.ID == id {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "You cannot disable the account you are currently using"})
			return
		}
		err = m.accounts.SetDisabled(id, true)
	case "enable":
		err = m.accounts.SetDisabled(id, false)
	case "reset_password":
		if err = m.accounts.ResetPassword(id, password); err == nil {
			// The password is durable now, so the old one must stop working even
			// if ending the old sign-ins then fails; the early return below used
			// to skip the invalidation and leave the memo serving the old password.
			m.forgetBasicCredentials()
			if revokeErr := m.revokeCredentials(id); revokeErr != nil {
				// The password is already changed; resetting again retries ending the old sign-ins.
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "The password was changed, but some old sign-ins could not be ended. Reset it again to retry"})
				return
			}
		}
	case "confirm_recovery":
		err = m.accounts.ConfirmRecovery(current.ID, password)
	default:
		http.Error(w, "Invalid account action", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeAccountError(w, err)
		return
	}
	m.forgetBasicCredentials()
	w.WriteHeader(http.StatusNoContent)
}

// revokeCredentials ends every sign-in an account holds, so a reset password
// leaves nobody signed in with the old one.
func (m *multiServer) revokeCredentials(account string) error {
	var first error
	if m.dropSessions != nil {
		first = m.dropSessions(account)
	}
	for _, revoke := range []func(string) error{
		func(id string) error {
			if m.androidDevices != nil {
				return m.androidDevices.revokeAccount(id)
			}
			return nil
		},
		func(id string) error {
			if m.jellyfinDevices != nil {
				return m.jellyfinDevices.revokeAccount(id)
			}
			return nil
		},
		func(id string) error {
			if m.devices != nil {
				return m.devices.revokeAccount(id)
			}
			return nil
		},
	} {
		if err := revoke(account); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func writeAccountError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, accounts.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "That username is already in use"})
	case errors.Is(err, accounts.ErrLastAdministrator):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "At least one enabled administrator is required"})
	case errors.Is(err, accounts.ErrAccountMustBeDisabled):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Disable this account before deleting it"})
	case errors.Is(err, accounts.ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Use a unique username, a password of 10 to 72 characters, and a valid role"})
	case errors.Is(err, accounts.ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Enter the current account password before disabling environment recovery"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not update the account"})
	}
}
