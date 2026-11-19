package web

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// A play request is answered by the server without Hermes, so Hermes never sees it and would not list it when the chat
// is opened again. The exchange is kept here and put back into the chat at the place where it happened, with its cards,
// so a song can be played again later.

const maximumPlaysPerChat = 40

type hermesPlay struct {
	At     int64        `json:"at"`
	Before int          `json:"before"` // how many of Hermes's own messages came first
	User   string       `json:"user"`
	Reply  string       `json:"reply"`
	Cards  []hermesCard `json:"cards"`
}

type hermesPlayStore struct {
	mu    sync.Mutex
	path  string
	chats map[string][]hermesPlay
}

func newHermesPlayStore(path string) *hermesPlayStore {
	s := &hermesPlayStore{path: path, chats: map[string][]hermesPlay{}}
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &s.chats)
		}
	}
	return s
}

func (s *hermesPlayStore) saveLocked() {
	if s.path == "" {
		return
	}
	data, err := json.Marshal(s.chats)
	if err != nil {
		return
	}
	temporary := s.path + ".tmp"
	if os.WriteFile(temporary, data, 0o600) == nil {
		_ = os.Rename(temporary, s.path)
	}
}

func (s *hermesPlayStore) add(chat string, play hermesPlay) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := append(s.chats[chat], play)
	if len(list) > maximumPlaysPerChat {
		list = list[len(list)-maximumPlaysPerChat:]
	}
	s.chats[chat] = list
	s.saveLocked()
}

func (s *hermesPlayStore) forget(chat string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.chats[chat]; ok {
		delete(s.chats, chat)
		s.saveLocked()
	}
}

// merge puts the saved play exchanges back among Hermes's own messages.
func (s *hermesPlayStore) merge(chat string, messages []hermesMessage) []hermesMessage {
	s.mu.Lock()
	plays := append([]hermesPlay(nil), s.chats[chat]...)
	s.mu.Unlock()
	if len(plays) == 0 {
		return messages
	}
	sort.SliceStable(plays, func(i, j int) bool { return plays[i].At < plays[j].At })
	out := append([]hermesMessage(nil), messages...)
	inserted := 0
	for _, play := range plays {
		at := play.Before + inserted
		if at > len(out) {
			at = len(out)
		}
		pair := []hermesMessage{{Role: "user", Text: play.User}, {Role: "assistant", Text: play.Reply, Cards: play.Cards}}
		out = append(out[:at], append(pair, out[at:]...)...)
		inserted += 2
	}
	return out
}

func (m *multiServer) hermesPlayStore() *hermesPlayStore {
	m.hermesPlaysOnce.Do(func() {
		path := ""
		if m.cfg.AudioState != "" {
			path = filepath.Join(filepath.Dir(m.cfg.AudioState), "hermes-plays.json")
		}
		m.hermesPlays = newHermesPlayStore(path)
	})
	return m.hermesPlays
}

// hermesMessageCount is how many messages Hermes lists for the chat now (best effort: 0 when it cannot be asked).
func (m *multiServer) hermesMessageCount(ctx context.Context, path string) int {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	response, err := hermesCall(ctx, m.hermesSettings(), "GET", path+"/messages", nil, "")
	if err != nil {
		return 0
	}
	defer response.Body.Close()
	var body struct {
		Data []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"data"`
	}
	if response.StatusCode != 200 || json.NewDecoder(response.Body).Decode(&body) != nil {
		return 0
	}
	count := 0
	for _, item := range body.Data {
		if text, _ := item.Content.(string); (item.Role == "user" || item.Role == "assistant") && text != "" {
			count++
		}
	}
	return count
}
