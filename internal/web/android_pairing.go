package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	androidPairingLifetime = 10 * time.Minute
	androidPairingMaximum  = 32
)

type androidPairing struct {
	DeviceName string
	Code       string
	Approval   [32]byte
	CreatedAt  time.Time
	Approved   bool
	Account    string
}

type androidPairingStore struct {
	mu      sync.Mutex
	now     func() time.Time
	pending map[[32]byte]androidPairing
}

func newAndroidPairingStore() *androidPairingStore {
	return &androidPairingStore{now: time.Now, pending: make(map[[32]byte]androidPairing)}
}

func (s *androidPairingStore) create(deviceName string) (id, code string, expires time.Time, ok bool) {
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" || len(deviceName) > 80 {
		return "", "", time.Time{}, false
	}
	idRaw := make([]byte, 32)
	codeRaw := make([]byte, 6)
	if _, err := rand.Read(idRaw); err != nil {
		return "", "", time.Time{}, false
	}
	if _, err := rand.Read(codeRaw); err != nil {
		return "", "", time.Time{}, false
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for i := range codeRaw {
		codeRaw[i] = alphabet[int(codeRaw[i])%len(alphabet)]
	}
	id = base64.RawURLEncoding.EncodeToString(idRaw)
	code = string(codeRaw)
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	if len(s.pending) >= androidPairingMaximum {
		// Drop the oldest request instead of refusing: the endpoint is open to
		// unpaired devices, so refusing would let one client fill the store and
		// lock every real television out.
		var oldestKey [32]byte
		var oldest time.Time
		for key, pairing := range s.pending {
			if oldest.IsZero() || pairing.CreatedAt.Before(oldest) {
				oldestKey, oldest = key, pairing.CreatedAt
			}
		}
		delete(s.pending, oldestKey)
	}
	s.pending[sha256.Sum256([]byte(id))] = androidPairing{DeviceName: deviceName, Code: code, CreatedAt: now}
	return id, code, now.Add(androidPairingLifetime), true
}

func (s *androidPairingStore) approvalToken(code string) (string, bool) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", false
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	for key, pairing := range s.pending {
		if subtle.ConstantTimeCompare([]byte(pairing.Code), []byte(strings.ToUpper(strings.TrimSpace(code)))) == 1 {
			pairing.Approval = sha256.Sum256([]byte(token))
			s.pending[key] = pairing
			return token, true
		}
	}
	return "", false
}

func (s *androidPairingStore) approve(code, token, account string) (string, bool) {
	if account == "" {
		return "", false
	}
	now := s.now().UTC()
	want := sha256.Sum256([]byte(strings.TrimSpace(token)))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	for key, pairing := range s.pending {
		if subtle.ConstantTimeCompare([]byte(pairing.Code), []byte(strings.ToUpper(strings.TrimSpace(code)))) == 1 && subtle.ConstantTimeCompare(pairing.Approval[:], want[:]) == 1 {
			pairing.Approved = true
			pairing.Account = account
			pairing.Approval = [32]byte{}
			s.pending[key] = pairing
			return pairing.DeviceName, true
		}
	}
	return "", false
}

func (s *androidPairingStore) exchange(id string, issue func(string, string) (map[string]string, error)) (map[string]string, string) {
	key := sha256.Sum256([]byte(strings.TrimSpace(id)))
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	pairing, ok := s.pending[key]
	if !ok {
		return nil, "expired"
	}
	if !pairing.Approved {
		return nil, "pending"
	}
	credential, err := issue(pairing.Account, pairing.DeviceName)
	if err != nil {
		return nil, "failed"
	}
	delete(s.pending, key)
	return credential, "approved"
}

func (s *androidPairingStore) pruneLocked(now time.Time) {
	for key, pairing := range s.pending {
		if now.Sub(pairing.CreatedAt) > androidPairingLifetime {
			delete(s.pending, key)
		}
	}
}

func (m *multiServer) androidPairingAPI(w http.ResponseWriter, r *http.Request) {
	if m.androidPairings == nil || m.androidDevices == nil {
		http.Error(w, "TV pairing requires persistent reading state", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	suffix := strings.TrimPrefix(r.URL.Path, "/api/android/v1/pairing")
	if suffix == "" && r.Method == http.MethodPost {
		var input struct {
			DeviceName string `json:"device_name"`
		}
		if decodeJSONRequest(w, r, 4<<10, &input, true) != nil {
			http.Error(w, "Invalid pairing request", http.StatusBadRequest)
			return
		}
		id, code, expires, ok := m.androidPairings.create(input.DeviceName)
		if !ok {
			http.Error(w, "Could not create pairing request", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"pairing_id": id, "user_code": code, "verification_uri": m.publicBase(r) + "/pair?code=" + code, "expires_at": expires})
		return
	}
	if strings.HasPrefix(suffix, "/") && r.Method == http.MethodGet {
		credential, state := m.androidPairings.exchange(strings.TrimPrefix(suffix, "/"), func(account, name string) (map[string]string, error) {
			token, deviceID, err := m.androidDevices.issue(account, name)
			if err != nil {
				return nil, err
			}
			return map[string]string{"device_id": deviceID, "device_token": token, "server_url": m.publicBase(r)}, nil
		})
		switch state {
		case "approved":
			writeJSON(w, http.StatusCreated, credential)
		case "pending":
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
		case "expired":
			http.Error(w, "Pairing request expired", http.StatusGone)
		default:
			http.Error(w, "Could not complete pairing", http.StatusServiceUnavailable)
		}
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (m *multiServer) androidPairingPage(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	data := map[string]string{"Code": code}
	status := http.StatusOK
	if r.Method == http.MethodGet {
		if token, ok := m.androidPairings.approvalToken(code); ok {
			data["Token"] = token
		} else if code != "" {
			data["Message"] = "That pairing code is invalid or expired."
		}
	} else if r.Method == http.MethodPost {
		if name, ok := m.androidPairings.approve(code, r.FormValue("approval"), m.readingAccount(r)); ok {
			data["Device"] = name
		} else {
			data["Message"] = "That approval expired. Enter the code again."
			status = http.StatusBadRequest
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'")
	w.WriteHeader(status)
	_ = androidPairingTemplate.Execute(w, data)
}

var androidPairingTemplate = template.Must(template.New("android-pairing").Parse(`<!doctype html><html lang="en" data-theme="dark"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><meta name="color-scheme" content="dark light"><title>Pair TV · Mochil</title><link rel="stylesheet" href="/assets/design-tokens.css"><link rel="stylesheet" href="/assets/login.css"></head><body><main><section><a class="wordmark" href="/" aria-label="Mochil"><span><img src="/assets/icons/icon-128.png" alt=""></span> Mochil</a>{{if .Device}}<div><h1>TV connected</h1><p class="intro">{{.Device}} may now open your Mochil libraries. You can close this page.</p></div>{{else}}<div><h1>Pair a TV</h1><p class="intro">Enter the code shown by Mochil on your television.</p></div><form method="post" action="/pair" autocomplete="off"><label>Pairing code<input name="code" value="{{.Code}}" required minlength="6" maxlength="6" autocapitalize="characters" autocomplete="off" spellcheck="false"></label>{{if .Token}}<input type="hidden" name="approval" value="{{.Token}}"><button type="submit">Connect TV</button>{{else}}<button type="submit" formmethod="get">Continue</button>{{end}}{{if .Message}}<p class="error" role="alert">{{.Message}}</p>{{end}}</form><small>Only approve a code currently visible on your own TV. Pairing creates a revocable device connection; it never shares your password.</small>{{end}}</section></main></body></html>`))
