package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// What the assistant has installed (plugins, skills, tool servers) comes from a small read-only helper that runs next
// to Hermes (tools/hermes-inventory). Hermes itself does not offer this list over its API.

const hermesInventoryPort = "8643"

// hermesInventoryAddress is the helper's address: the configured one, or the Hermes host on the helper's usual port.
func hermesInventoryAddress(configured, hermesAddress string) (string, bool) {
	raw := strings.TrimSpace(configured)
	if raw == "" {
		base, err := url.Parse(strings.TrimSpace(hermesAddress))
		if err != nil || base.Hostname() == "" {
			return "", false
		}
		raw = base.Scheme + "://" + net.JoinHostPort(base.Hostname(), hermesInventoryPort) + "/inventory"
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", false
	}
	return parsed.String(), true
}

func (m *multiServer) hermesInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	settings := m.hermesSettings()
	address, ok := hermesInventoryAddress(settings.InventoryURL, settings.URL)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": "Hermes is not set up"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": "The helper address is not valid"})
		return
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil {
			response.Body.Close()
		}
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": "The list helper is not running. See tools/hermes-inventory/README.md."})
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": "The helper's answer was cut short"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append([]byte(`{"available":true,"inventory":`), append(body, '}')...))
}

// hermesHelperCall sends one request to the list helper and returns its answer, or a reason it could not.
func (m *multiServer) hermesHelperCall(r *http.Request, method, path string, body []byte, authorize bool, timeout time.Duration) ([]byte, string) {
	settings := m.hermesSettings()
	address, ok := hermesInventoryAddress(settings.InventoryURL, settings.URL)
	if !ok {
		return nil, "Hermes is not set up"
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(address, "/inventory")+path, bytes.NewReader(body))
	if err != nil {
		return nil, "The helper address is not valid"
	}
	if authorize {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(settings.Key))
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		return nil, "The helper is not running. See tools/hermes-inventory/README.md."
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode == http.StatusForbidden {
		return nil, "The helper refused: it could not match the Hermes key"
	}
	if response.StatusCode != http.StatusOK {
		return nil, "The helper could not answer"
	}
	return data, ""
}

// hermesCatalog searches the reviewed catalog of plugins or skills (read only).
func (m *multiServer) hermesCatalog(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if r.Method != http.MethodGet || (kind != "plugins" && kind != "skills") {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	data, reason := m.hermesHelperCall(r, http.MethodGet, "/catalog/"+kind+"?q="+url.QueryEscape(strings.TrimSpace(r.URL.Query().Get("q"))), nil, false, 90*time.Second)
	if reason != "" {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "reason": reason})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append([]byte(`{"available":true,"result":`), append(data, '}')...))
}

// hermesInstall installs one reviewed plugin or skill, left disabled. The helper checks the entry again and the key.
func (m *multiServer) hermesInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	if err := decodeJSONRequest(w, r, 2<<10, &input, true); err != nil || (input.Kind != "plugin" && input.Kind != "skill") || strings.TrimSpace(input.Name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Say what to install"})
		return
	}
	body, _ := json.Marshal(map[string]string{"kind": input.Kind, "name": strings.TrimSpace(input.Name)})
	data, reason := m.hermesHelperCall(r, http.MethodPost, "/install", body, true, 5*time.Minute)
	if reason != "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "message": reason})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
