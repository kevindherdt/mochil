package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mochil/internal/appconfig"
)

// The Hermes module is a thin, administrator-only pass-through to a Hermes Agent API server. The key lives in the
// server config and never reaches a browser or an app. Conversations are Hermes sessions: Hermes stores them, and
// Mochil lists, opens, renames and deletes the ones that belong to the signed-in account. A chat is a Hermes
// session whose id is "opds-<account>-<hash>.<chat id>", so one account cannot open another's chat.

const maximumHermesRequestBytes = 256 << 10

var (
	hermesChatIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{7,47}$`)
	hermesNotAlnum      = regexp.MustCompile(`[^a-z0-9]+`)
)

type hermesChatRequest struct {
	Session string `json:"session"`
	Message string `json:"message"`
	// Spoken is true while replies are being read aloud; the answer is then written to be heard, not read.
	Spoken bool `json:"spoken"`
	// Interrupted is true when the person talked over her previous spoken reply, which was cut off.
	Interrupted bool `json:"interrupted"`
}

const hermesSpokenStyle = "Your reply will be read aloud. Answer the way you would speak: short, natural sentences, usually two to four of them unless asked for more. " +
	"No markdown, lists, headings, emoji, links or stage directions in asterisks."

// hermesSettings reads the Hermes block from the live config file, so changes saved in Settings apply at once without
// a restart. The values the server started with are the fallback.
func (m *multiServer) hermesSettings() appconfig.Hermes {
	if m.cfg.ConfigJSON != nil {
		if data, err := m.cfg.ConfigJSON(); err == nil {
			var document struct {
				Hermes *appconfig.Hermes `json:"hermes"`
			}
			if json.Unmarshal(data, &document) == nil && document.Hermes != nil {
				return *document.Hermes
			}
		}
	}
	return m.cfg.Hermes
}

func hermesConfigured(settings appconfig.Hermes) bool {
	return strings.TrimSpace(settings.URL) != "" && strings.TrimSpace(settings.Key) != ""
}

func hermesURL(settings appconfig.Hermes, path string) (string, bool) {
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(settings.URL), "/"))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", false
	}
	return base.String() + path, true
}

func hermesAssistantName(settings appconfig.Hermes) string {
	if name := strings.TrimSpace(settings.Name); name != "" {
		return name
	}
	return "Hermes"
}

// hermesPersona is the standing guidance sent with every message, with {name} replaced by the assistant's name.
func hermesPersona(settings appconfig.Hermes) string {
	persona := strings.TrimSpace(settings.Persona)
	if persona == "" {
		return ""
	}
	return strings.ReplaceAll(persona, "{name}", hermesAssistantName(settings))
}

// hermesSessionPrefix names the Hermes sessions of one account.
//
// The account part is a sanitised name plus a short hash of the exact account ID,
// ended by '.', which no account name (alphanumeric only) and no chat id
// ([a-z0-9-]) can contain. A plain "-" separator made "opds-sam-" a prefix of
// "opds-samuel-<chat>", so one account could read another's chats; the truncation
// collision between long names is covered by the hash too.
func (m *multiServer) hermesSessionPrefix(r *http.Request) string {
	account := strings.ToLower(m.readingAccount(r))
	name := hermesNotAlnum.ReplaceAllString(account, "")
	if len(name) > 12 {
		name = name[:12]
	}
	sum := sha256.Sum256([]byte(account))
	return "opds-" + name + "-" + hex.EncodeToString(sum[:4]) + "."
}

func (m *multiServer) apiHermes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/hermes"), "/")
	switch {
	case rest == "status":
		m.hermesStatus(w, r)
	case rest == "test":
		m.hermesTest(w, r)
	case rest == "chat":
		m.hermesChat(w, r)
	case rest == "polish":
		m.hermesPolish(w, r)
	case rest == "music-names" || strings.HasPrefix(rest, "music-names/"):
		m.hermesMusicNames(w, r)
	case rest == "catalog":
		m.hermesCatalog(w, r)
	case rest == "install":
		m.hermesInstall(w, r)
	case rest == "inventory":
		m.hermesInventory(w, r)
	case rest == "memories":
		m.hermesMemories(w, r)
	case rest == "remember":
		m.hermesRemember(w, r)
	case rest == "chats":
		m.hermesChatList(w, r)
	case strings.HasPrefix(rest, "chats/"):
		m.hermesChatOne(w, r, strings.TrimPrefix(rest, "chats/"))
	default:
		http.NotFound(w, r)
	}
}

// hermesCall sends one authenticated request to Hermes.
func hermesCall(ctx context.Context, settings appconfig.Hermes, method, path string, body []byte, accept string) (*http.Response, error) {
	address, ok := hermesURL(settings, path)
	if !ok {
		return nil, context.Canceled
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, address, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(settings.Key))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	return (&http.Client{}).Do(request)
}

// hermesProbe says whether a Hermes server answers, and whether it accepts the key.
func hermesProbe(ctx context.Context, settings appconfig.Hermes) (reachable, authorized bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	health, err := hermesCall(ctx, settings, http.MethodGet, "/health", nil, "")
	if err != nil {
		return false, false
	}
	health.Body.Close()
	models, err := hermesCall(ctx, settings, http.MethodGet, "/v1/models", nil, "")
	if err != nil {
		return true, false
	}
	models.Body.Close()
	return true, models.StatusCode == http.StatusOK
}

// hermesStatus says whether Hermes is set up and answering, without exposing the address or key.
func (m *multiServer) hermesStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	settings := m.hermesSettings()
	status := map[string]any{"configured": hermesConfigured(settings), "reachable": false, "name": hermesAssistantName(settings)}
	_, _, status["memory"] = hermesMemoryBase(settings)
	features := map[string]bool{}
	for _, name := range appconfig.HermesFeatures {
		features[name] = settings.Feature(name)
	}
	status["features"] = features
	if status["configured"] == true {
		reachable, authorized := hermesProbe(r.Context(), settings)
		status["reachable"] = reachable && authorized
		status["authorized"] = authorized
	}
	writeJSON(w, http.StatusOK, status)
}

// hermesTest checks an address and key typed in Settings (a blank key means the saved one) without saving them.
func (m *multiServer) hermesTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		URL string `json:"url"`
		Key string `json:"key"`
	}
	if err := decodeJSONRequest(w, r, 4<<10, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	settings := m.hermesSettings()
	if strings.TrimSpace(input.URL) != "" {
		settings.URL = input.URL
	}
	if strings.TrimSpace(input.Key) != "" {
		settings.Key = input.Key
	}
	if _, ok := hermesURL(settings, ""); !ok || strings.TrimSpace(settings.URL) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "message": "Enter the Hermes address, for example http://192.0.2.10:8642"})
		return
	}
	if strings.TrimSpace(settings.Key) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "message": "Enter the Hermes API key"})
		return
	}
	reachable, authorized := hermesProbe(r.Context(), settings)
	switch {
	case !reachable:
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "message": "Hermes did not answer at that address"})
	case !authorized:
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "message": "Hermes answered but refused the key"})
	default:
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Connected to Hermes"})
	}
}

type hermesChatSummary struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Updated float64 `json:"updated"`
	Count   int     `json:"count"`
}

// hermesChatList returns this account's Hermes sessions, newest first.
func (m *multiServer) hermesChatList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	settings := m.hermesSettings()
	if !hermesConfigured(settings) {
		writeJSON(w, http.StatusOK, map[string]any{"chats": []hermesChatSummary{}})
		return
	}
	prefix := m.hermesSessionPrefix(r)
	chats := []hermesChatSummary{}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	for offset := 0; offset < 1000; offset += 100 {
		response, err := hermesCall(ctx, settings, http.MethodGet, "/api/sessions?limit=100&offset="+strconv.Itoa(offset), nil, "")
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes is not reachable"})
			return
		}
		var page struct {
			Data []struct {
				ID           string  `json:"id"`
				Title        string  `json:"title"`
				Preview      string  `json:"preview"`
				MessageCount int     `json:"message_count"`
				LastActive   float64 `json:"last_active"`
				StartedAt    float64 `json:"started_at"`
			} `json:"data"`
			HasMore bool `json:"has_more"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&page)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || decodeErr != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes refused the request"})
			return
		}
		for _, session := range page.Data {
			id, ok := strings.CutPrefix(session.ID, prefix)
			if !ok || !hermesChatIDPattern.MatchString(id) || session.MessageCount == 0 {
				continue
			}
			title := strings.TrimSpace(session.Title)
			if title == "" {
				title = strings.TrimSpace(session.Preview)
			}
			if title == "" {
				title = "New chat"
			}
			if len([]rune(title)) > 60 {
				title = string([]rune(title)[:59]) + "…"
			}
			updated := session.LastActive
			if updated == 0 {
				updated = session.StartedAt
			}
			chats = append(chats, hermesChatSummary{ID: id, Title: title, Updated: updated, Count: session.MessageCount})
		}
		if !page.HasMore {
			break
		}
	}
	sort.Slice(chats, func(i, j int) bool { return chats[i].Updated > chats[j].Updated })
	writeJSON(w, http.StatusOK, map[string]any{"chats": chats})
}

type hermesMessage struct {
	Role  string       `json:"role"`
	Text  string       `json:"text"`
	Cards []hermesCard `json:"cards,omitempty"`
}

// hermesChatOne opens, renames or deletes one chat.
func (m *multiServer) hermesChatOne(w http.ResponseWriter, r *http.Request, id string) {
	settings := m.hermesSettings()
	if !hermesConfigured(settings) || !hermesChatIDPattern.MatchString(id) {
		http.NotFound(w, r)
		return
	}
	path := "/api/sessions/" + url.PathEscape(m.hermesSessionPrefix(r)+id)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	switch r.Method {
	case http.MethodGet:
		response, err := hermesCall(ctx, settings, http.MethodGet, path+"/messages", nil, "")
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes is not reachable"})
			return
		}
		defer response.Body.Close()
		if response.StatusCode == http.StatusNotFound {
			writeJSON(w, http.StatusOK, map[string]any{"messages": m.hermesPlayStore().merge(m.hermesSessionPrefix(r)+id, []hermesMessage{})})
			return
		}
		var body struct {
			Data []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"data"`
		}
		if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&body) != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes refused the request"})
			return
		}
		messages := []hermesMessage{}
		for _, item := range body.Data {
			text, _ := item.Content.(string)
			if (item.Role != "user" && item.Role != "assistant") || strings.TrimSpace(text) == "" {
				continue
			}
			messages = append(messages, hermesMessage{Role: item.Role, Text: text})
		}
		writeJSON(w, http.StatusOK, map[string]any{"messages": m.hermesPlayStore().merge(m.hermesSessionPrefix(r)+id, messages)})
	case http.MethodPatch:
		var input struct {
			Title string `json:"title"`
		}
		title := ""
		if err := decodeJSONRequest(w, r, 4<<10, &input, true); err == nil {
			title = strings.TrimSpace(input.Title)
		}
		if title == "" || len(title) > 200 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A title is required"})
			return
		}
		body, _ := json.Marshal(map[string]string{"title": title})
		hermesRelay(w, ctx, settings, http.MethodPatch, path, body)
	case http.MethodDelete:
		m.hermesPlayStore().forget(m.hermesSessionPrefix(r) + id)
		hermesRelay(w, ctx, settings, http.MethodDelete, path, nil)
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// hermesRelay performs a Hermes call and answers with a small JSON result.
func hermesRelay(w http.ResponseWriter, ctx context.Context, settings appconfig.Hermes, method, path string, body []byte) {
	response, err := hermesCall(ctx, settings, method, path, body, "")
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes is not reachable"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 && response.StatusCode != http.StatusNotFound {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes refused the request"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// hermesChat sends one message into a Hermes session and relays the streamed reply (server-sent events). The request
// to Hermes runs to the end even if the browser goes away, so the reply is saved in the session and any device can
// open it afterwards.
func (m *multiServer) hermesChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	settings := m.hermesSettings()
	if !hermesConfigured(settings) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Hermes is not set up. Open Settings, Hermes, and enter the address and key."})
		return
	}
	var input hermesChatRequest
	if err := decodeJSONRequest(w, r, maximumHermesRequestBytes, &input, true); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid chat request"})
		return
	}
	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" || !hermesChatIDPattern.MatchString(input.Session) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A message and a session id are required"})
		return
	}
	model := strings.TrimSpace(settings.Model)
	if model == "" {
		model = "hermes-agent"
	}
	messages := []map[string]string{}
	guidance := hermesPersona(settings)
	if input.Spoken {
		guidance = strings.TrimSpace(guidance + "\n\n" + hermesSpokenStyle)
	}
	if !settings.Feature("home") {
		guidance = strings.TrimSpace(guidance + "\n\nHome control is switched off in Mochil: do not use any Home Assistant, TV or device tool, and if asked, say it is switched off in Settings.")
	}
	if !settings.Feature("radio") {
		guidance = strings.TrimSpace(guidance + "\n\nRadio and speakers are switched off in Mochil: do not use play_radio or any speaker, volume or music tool, and if asked, say it is switched off in Settings.")
	}
	if input.Interrupted {
		guidance = strings.TrimSpace(guidance + "\n\nThe person spoke over your previous reply, so it was cut off before they heard all of it. Do not repeat it; react naturally to what they say now.")
	}
	if cards, reply := m.hermesPlayLookup(m.readingAccount(r), input.Message, settings); len(cards) > 0 {
		sessionKey := m.hermesSessionPrefix(r) + input.Session
		before := m.hermesMessageCount(r.Context(), "/api/sessions/"+url.PathEscape(sessionKey))
		m.hermesPlayStore().add(sessionKey, hermesPlay{At: time.Now().UnixNano(), Before: before, User: input.Message, Reply: reply, Cards: cards})
		stream := newHermesStreamWriter(w)
		stream.begin()
		stream.write(hermesCardsEvent(cards))
		chunk, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": reply}}}})
		stream.write([]byte("data: " + string(chunk) + "\n\ndata: [DONE]\n\n"))
		return
	}
	if guidance != "" {
		messages = append(messages, map[string]string{"role": "system", "content": guidance})
	}
	messages = append(messages, map[string]string{"role": "user", "content": input.Message})
	body, _ := json.Marshal(map[string]any{"model": model, "stream": true, "messages": messages})

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 15*time.Minute)
	defer cancel()
	address, ok := hermesURL(settings, "/v1/chat/completions")
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "The Hermes address is not valid"})
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Hermes request failed", http.StatusInternalServerError)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(settings.Key))
	request.Header.Set("X-Hermes-Session-Id", m.hermesSessionPrefix(r)+input.Session)
	stream := newHermesStreamWriter(w)
	done := make(chan hermesUpstream, 1)
	go func() {
		response, err := (&http.Client{}).Do(request)
		done <- hermesUpstream{response: response, err: err}
	}()
	upstream := waitForHermes(stream, done)
	if upstream.err != nil {
		if stream.open {
			stream.fail("Hermes is not reachable")
		} else {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Hermes is not reachable"})
		}
		return
	}
	response := upstream.response
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 2<<10))
		if stream.open {
			stream.fail(fmt.Sprintf("Hermes refused the request (%d)", response.StatusCode))
		} else {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Hermes refused the request", "status": response.StatusCode, "detail": strings.TrimSpace(string(message))})
		}
		return
	}
	stream.begin()
	pumpHermes(stream, response.Body)
}
