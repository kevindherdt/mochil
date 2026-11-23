package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"mochil/internal/appconfig"
)

var (
	errHermesUnreachable = errors.New("the assistant endpoint did not answer")
	errHermesFailed      = errors.New("the assistant endpoint could not answer")
	errHelperNotSet      = errors.New("no helper model is set")
)

// hermesAsk sends one question to the helper model and returns its whole answer. It talks to a plain language model
// (settings.HelperURL, OpenAI-style), never to the Hermes agent: the agent has tools and a persona, and a small model
// given a one-line task once looped through Home Assistant calls for an hour. A plain model has nothing to wander into,
// and no long agent prompt to read first. It keeps no history, so helpers never show up among the person's chats.
// The caller owns the timeout.
// helperTarget says where a one-shot question goes. A dedicated helper model wins when one is configured; otherwise the
// question goes to the Hermes gateway itself — the same address, key and model the chat uses — so the one-shots follow
// whatever the profile runs by default instead of needing a second endpoint kept in step with it.
func helperTarget(settings appconfig.Hermes) (base, model, key string, ok bool) {
	if settings.HelperConfigured() {
		address := strings.TrimRight(strings.TrimSpace(settings.HelperURL), "/")
		if !strings.HasSuffix(address, "/v1") {
			address += "/v1"
		}
		return address, strings.TrimSpace(settings.HelperModel), "", true
	}
	if !settings.AskConfigured() {
		return "", "", "", false
	}
	return strings.TrimRight(strings.TrimSpace(settings.URL), "/") + "/v1", strings.TrimSpace(settings.Model), strings.TrimSpace(settings.Key), true
}

func hermesAsk(ctx context.Context, settings appconfig.Hermes, system, user string) (string, error) {
	base, model, key, ok := helperTarget(settings)
	if !ok {
		return "", errHelperNotSet
	}
	request, _ := json.Marshal(map[string]any{
		"model": model, "stream": false,
		"reasoning_effort": "none", // thinking models: just answer (ignored by servers that do not know it)
		"messages":         []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": user}},
	})
	call, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(request))
	if err != nil {
		return "", errors.Join(errHermesUnreachable, err)
	}
	call.Header.Set("Content-Type", "application/json")
	if key != "" {
		call.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := (&http.Client{}).Do(call)
	if err != nil {
		return "", errors.Join(errHermesUnreachable, err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	var answer struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if response.StatusCode != http.StatusOK || json.Unmarshal(raw, &answer) != nil || len(answer.Choices) == 0 {
		return "", errHermesFailed
	}
	return answer.Choices[0].Message.Content, nil
}

// Hermes runs one model at a time, and a small local model works through requests one after another, so helpers take
// turns: a second request would only queue behind the first and make both look stuck. A person is told it is busy;
// a background job waits its turn.
func (m *multiServer) tryHermesTurn() bool { return m.hermesTurn.TryLock() }
func (m *multiServer) endHermesTurn()      { m.hermesTurn.Unlock() }

func (m *multiServer) waitHermesTurn(ctx context.Context) bool {
	for !m.hermesTurn.TryLock() {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(500 * time.Millisecond):
		}
	}
	return true
}
