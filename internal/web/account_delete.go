package web

import (
	"fmt"
	"net/http"
)

// deleteAccount is deliberately disable-first and retryable. Personal stores
// are purged before the credential record; if one durable write fails the
// account stays disabled and the administrator can safely retry.
func (m *multiServer) deleteAccount(w http.ResponseWriter, r *http.Request, id, confirmation string) {
	current, _ := identityFromRequest(r)
	if current.ID == id {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "You cannot delete the account you are currently using"})
		return
	}
	m.accountMutationMu.Lock()
	defer m.accountMutationMu.Unlock()
	summary, found := m.accountSummary(id)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Account not found"})
		return
	}
	if !summary.Disabled {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Disable this account before deleting it"})
		return
	}
	if confirmation != summary.Username {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter the exact username to confirm permanent deletion"})
		return
	}
	if id == m.cfg.Username && m.accountStore().RecoveryEnabled() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Finish account migration before deleting the original administrator"})
		return
	}
	if err := m.purgeAccountData(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Deletion was interrupted. The account remains disabled; retry to finish deleting its data"})
		return
	}
	if err := m.accountStore().DeleteDisabled(id); err != nil {
		writeAccountError(w, err)
		return
	}
	m.forgetBasicCredentials()
	w.WriteHeader(http.StatusNoContent)
}

func (m *multiServer) purgeAccountData(account string) error {
	steps := []struct {
		name string
		run  func() error
	}{
		{"Android devices", func() error {
			if m.androidDevices != nil {
				return m.androidDevices.revokeAccount(account)
			}
			return nil
		}},
		{"Jellyfin devices", func() error {
			if m.jellyfinDevices != nil {
				return m.jellyfinDevices.revokeAccount(account)
			}
			return nil
		}},
		{"KOReader devices", func() error {
			if m.devices != nil {
				return m.devices.revokeAccount(account)
			}
			return nil
		}},
		{"reading", func() error {
			if m.reading != nil {
				return m.reading.DeleteAccount(account)
			}
			return nil
		}},
		{"annotations", func() error {
			if m.annotations != nil {
				_, err := m.annotations.DeleteAccount(account)
				return err
			}
			return nil
		}},
		{"audio progress", func() error {
			if m.audioProgress != nil {
				return m.audioProgress.DeleteAccount(account)
			}
			return nil
		}},
		{"video progress", func() error {
			if m.videoProgress != nil {
				return m.videoProgress.DeleteAccount(account)
			}
			return nil
		}},
		{"queue", func() error {
			if m.audioQueue != nil {
				return m.audioQueue.DeleteAccount(account)
			}
			return nil
		}},
		{"favorites", func() error {
			if m.audioFavorites != nil {
				return m.audioFavorites.DeleteAccount(account)
			}
			return nil
		}},
		{"playlists", func() error {
			if m.audioPlaylists != nil {
				return m.audioPlaylists.DeleteAccount(account)
			}
			return nil
		}},
		{"collections", func() error {
			if m.collections != nil {
				return m.collections.deleteAccount(account)
			}
			return nil
		}},
		{"wiki", func() error {
			if m.wiki != nil {
				return m.wiki.DeleteAccount(account)
			}
			return nil
		}},
		{"external library", func() error {
			if m.externalLibrary != nil {
				return m.externalLibrary.DeleteAccount(account)
			}
			return nil
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("purge %s: %w", step.name, err)
		}
	}
	return nil
}
