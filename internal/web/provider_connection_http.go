package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"mochil/internal/enrichment"
)

func (m *multiServer) apiProviderConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Provider string `json:"provider"`
		Token    string `json:"token"`
	}
	if err := decodeJSONRequest(w, r, 8<<10, &input, true); err != nil {
		http.Error(w, "Invalid provider test", http.StatusBadRequest)
		return
	}
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.Token = strings.TrimSpace(input.Token)
	if len(input.Token) > 4096 || (input.Provider != "hardcover" && input.Provider != "tmdb") {
		http.Error(w, "Invalid provider test", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var err error
	if input.Provider == "hardcover" {
		provider := m.hardcover
		if input.Token != "" {
			provider = enrichment.NewHardcover(input.Token, nil)
		}
		err = provider.Check(ctx)
	} else {
		provider := m.tmdb
		if input.Token != "" {
			provider = enrichment.NewTMDB(input.Token, nil)
		}
		err = provider.Check(ctx)
	}
	if err != nil {
		status, message := providerConnectionError(err)
		writeJSON(w, status, map[string]any{"connected": false, "provider": input.Provider, "message": message, "libraries": m.providerLibraries(input.Provider)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"connected": true, "provider": input.Provider, "message": "Connection verified", "libraries": m.providerLibraries(input.Provider)})
}

func providerConnectionError(err error) (int, string) {
	switch {
	case errors.Is(err, enrichment.ErrProviderNotConnected):
		return http.StatusConflict, "No saved token. Paste one here or save it first."
	case errors.Is(err, enrichment.ErrProviderRejected):
		return http.StatusUnauthorized, "The token was rejected or lacks the required access."
	case errors.Is(err, enrichment.ErrProviderRateLimited):
		return http.StatusTooManyRequests, "The provider rate limit was reached. Try again later."
	default:
		return http.StatusBadGateway, "The provider is temporarily unavailable. Try again."
	}
}

func (m *multiServer) providerLibraries(provider string) []map[string]string {
	result := make([]map[string]string, 0)
	for _, library := range m.libraries {
		if library.MetadataProvider == provider {
			result = append(result, map[string]string{"id": library.ID, "name": library.Name})
		}
	}
	return result
}
