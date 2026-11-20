package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mochil/internal/appconfig"
)

// What she remembers lives in Hindsight, a separate memory server that Hermes writes to after each turn. The chat only
// reads it (a list) and adds to it on request ("Remember this"). The server address is never sent to the browser.
const (
	maximumRememberCharacters = 4000
	hermesMemoryPage          = 50
)

var hindsightBankPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func hermesMemoryBase(settings appconfig.Hermes) (base, bank string, ok bool) {
	bank = strings.TrimSpace(settings.MemoryBank)
	if !settings.Feature("memory") || bank == "" || !hindsightBankPattern.MatchString(bank) {
		return "", "", false
	}
	raw := strings.TrimRight(strings.TrimSpace(settings.MemoryURL), "/")
	if raw == "" {
		raw = "http://hindsight:8888"
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", "", false
	}
	return parsed.String() + "/v1/default/banks/" + url.PathEscape(bank), bank, true
}

type hermesMemory struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Kind string `json:"kind,omitempty"`
	Date string `json:"date,omitempty"`
}

// cleanMemoryText drops the " | Where: ... | Involving: ..." detail Hindsight appends to a fact, which reads as noise in a list.
func cleanMemoryText(text string) string {
	if at := strings.Index(text, " | "); at > 0 {
		text = text[:at]
	}
	return strings.TrimSpace(text)
}

func (m *multiServer) hermesMemories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	base, _, ok := hermesMemoryBase(m.hermesSettings())
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "items": []hermesMemory{}})
		return
	}
	query := url.Values{}
	query.Set("limit", strconv.Itoa(hermesMemoryPage))
	if offset, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && offset > 0 {
		query.Set("offset", strconv.Itoa(offset))
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		query.Set("q", q)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/memories/list?"+query.Encode(), nil)
	if err != nil {
		http.Error(w, "Memory request failed", http.StatusInternalServerError)
		return
	}
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"enabled": true, "error": "The memory server is not reachable"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "items": []hermesMemory{}, "total": 0})
		return
	}
	if response.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusBadGateway, map[string]any{"enabled": true, "error": "The memory server refused the request"})
		return
	}
	var upstream struct {
		Items []struct {
			ID       string `json:"id"`
			Text     string `json:"text"`
			Kind     string `json:"fact_type"`
			Date     string `json:"date"`
			Mentions string `json:"mentioned_at"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&upstream); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"enabled": true, "error": "The memory server answered in an unexpected way"})
		return
	}
	items := make([]hermesMemory, 0, len(upstream.Items))
	seen := map[string]bool{}
	for _, item := range upstream.Items {
		text := cleanMemoryText(item.Text)
		key := strings.ToLower(strings.TrimRight(text, " .!"))
		if text == "" || seen[key] {
			continue
		}
		seen[key] = true
		date := item.Mentions
		if date == "" {
			date = item.Date
		}
		items = append(items, hermesMemory{ID: item.ID, Text: text, Kind: item.Kind, Date: date})
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "items": items, "total": upstream.Total})
}

func (m *multiServer) hermesRemember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	base, _, ok := hermesMemoryBase(m.hermesSettings())
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Memory is not set up"})
		return
	}
	var input struct {
		Text string `json:"text"`
	}
	if err := decodeJSONRequest(w, r, 64<<10, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	text := strings.TrimSpace(input.Text)
	if text == "" || len([]rune(text)) > maximumRememberCharacters {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Send between 1 and 4000 characters to remember"})
		return
	}
	body, _ := json.Marshal(map[string]any{
		"async": true,
		"items": []map[string]any{{"content": text, "context": "Kevin asked her to remember this", "tags": []string{"remembered"}}},
	})
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/memories", bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Memory request failed", http.StatusInternalServerError)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{}).Do(request)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "The memory server is not reachable"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "The memory server refused it"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
