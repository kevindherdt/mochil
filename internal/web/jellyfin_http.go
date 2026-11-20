package web

import (
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"mochil/internal/accounts"
	"mochil/internal/audio"
	"mochil/internal/jellyfin"
	"mochil/internal/video"
)

func isJellyfinPath(value string) bool {
	for _, prefix := range []string{"/System/", "/Users/AuthenticateByName", "/Users/", "/Library/VirtualFolders", "/Items", "/Videos/", "/Audio/", "/Sessions", "/Shows/", "/UserViews", "/Branding/", "/QuickConnect/", "/DisplayPreferences/", "/Playback/"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
func (m *multiServer) jellyfinServerID() string { return jellyfin.ID("server", m.cfg.PublicURL) }
func jellyfinRequestToken(r *http.Request) string {
	if value := r.Header.Get("X-Emby-Token"); value != "" {
		return value
	}
	if value := r.URL.Query().Get("api_key"); value != "" {
		return value
	}
	header := r.Header.Get("Authorization")
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		for _, key := range []string{"Token=", "token="} {
			if index := strings.Index(part, key); index >= 0 {
				return strings.Trim(strings.TrimSpace(part[index+len(key):]), `"`)
			}
		}
	}
	return ""
}
func (m *multiServer) jellyfinIdentity(r *http.Request) (accounts.Identity, bool) {
	if m.jellyfinDevices == nil {
		return accounts.Identity{}, false
	}
	account, _, ok := m.jellyfinDevices.authenticate(jellyfinRequestToken(r))
	if !ok {
		return accounts.Identity{}, false
	}
	if m.accounts != nil {
		return m.accounts.IdentityByID(account)
	}
	return accounts.Identity{ID: account, Username: account, Role: accounts.RoleAdministrator}, true
}

func (m *multiServer) jellyfinUser(identity accounts.Identity) map[string]any {
	user := jellyfin.NewUser(identity.Username, m.jellyfinServerID(), true)
	user["Id"] = jellyfin.ID("user", identity.ID)
	if policy, ok := user["Policy"].(map[string]any); ok {
		policy["IsAdministrator"] = identity.Role == accounts.RoleAdministrator
	}
	return user
}

func (m *multiServer) jellyfin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Application-Error-Code", "")
	if r.URL.Path == "/System/Info/Public" {
		writeJSON(w, http.StatusOK, map[string]any{"LocalAddress": m.cfg.PublicURL, "ServerName": "Mochil", "Version": "0.1.0", "ProductName": "Mochil", "OperatingSystem": "Linux", "Id": m.jellyfinServerID(), "StartupWizardCompleted": true})
		return
	}
	if r.URL.Path == "/System/Ping" {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Mochil"))
		return
	}
	if r.URL.Path == "/QuickConnect/Enabled" {
		writeJSON(w, http.StatusOK, false)
		return
	}
	if r.URL.Path == "/Branding/Configuration" {
		writeJSON(w, http.StatusOK, map[string]any{"LoginDisclaimer": "", "CustomCss": ""})
		return
	}
	if r.URL.Path == "/Users/Public" {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	if r.URL.Path == "/Users/AuthenticateByName" && r.Method == http.MethodPost {
		var credentials struct {
			Username string `json:"Username"`
			Pw       string `json:"Pw"`
			Password string `json:"Password"`
		}
		if decodeJSONRequest(w, r, 16<<10, &credentials, false) != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		password := credentials.Pw
		if password == "" {
			password = credentials.Password
		}
		remote := requestAddress(r)
		if m.jellyfinLogins.blocked(remote) {
			w.Header().Set("Retry-After", "900")
			http.Error(w, "Too many attempts", http.StatusTooManyRequests)
			return
		}
		identity, ok := m.authenticateAccount(credentials.Username, password)
		if !ok {
			m.jellyfinLogins.fail(remote)
		} else {
			m.jellyfinLogins.clear(remote)
		}
		if !ok || m.jellyfinDevices == nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		deviceName := strings.TrimSpace(r.Header.Get("User-Agent"))
		if deviceName == "" || len(deviceName) > 80 {
			deviceName = "Jellyfin client"
		}
		token, deviceID, err := m.jellyfinDevices.issue(identity.ID, deviceName)
		if err != nil {
			http.Error(w, "Could not create Jellyfin session", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, map[string]any{"User": m.jellyfinUser(identity), "SessionInfo": map[string]any{"Id": deviceID, "UserId": jellyfin.ID("user", identity.ID)}, "AccessToken": token, "ServerId": m.jellyfinServerID()})
		return
	}
	identity, ok := m.jellyfinIdentity(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `MediaBrowser Token=""`)
		http.Error(w, "Jellyfin authentication required", http.StatusUnauthorized)
		return
	}
	r = r.WithContext(contextWithIdentity(r.Context(), identity))
	switch {
	case r.URL.Path == "/System/Ping":
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Mochil"))
	case r.URL.Path == "/System/Info":
		writeJSON(w, http.StatusOK, map[string]any{"ServerName": "Mochil", "Version": "0.1.0", "ProductName": "Mochil", "OperatingSystem": "Linux", "Id": m.jellyfinServerID()})
	case r.URL.Path == "/System/Endpoint":
		writeJSON(w, http.StatusOK, map[string]any{"IsLocal": false, "IsInNetwork": false})
	case r.URL.Path == "/Branding/Configuration":
		writeJSON(w, http.StatusOK, map[string]any{"LoginDisclaimer": "", "CustomCss": ""})
	case r.URL.Path == "/QuickConnect/Enabled":
		writeJSON(w, http.StatusOK, false)
	case r.URL.Path == "/Playback/BitrateTest":
		size := queryInt(r, "size")
		if size < 0 {
			size = 0
		}
		if size > 1<<20 {
			size = 1 << 20
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(make([]byte, size))
	case r.URL.Path == "/Users/Me":
		writeJSON(w, http.StatusOK, m.jellyfinUser(identity))
	case r.URL.Path == "/Library/VirtualFolders":
		views := m.jellyfinLibraries()
		writeJSON(w, http.StatusOK, views)
	case strings.HasSuffix(r.URL.Path, "/Views"):
		writeJSON(w, http.StatusOK, map[string]any{"Items": m.jellyfinLibraries(), "TotalRecordCount": len(m.jellyfinLibraries())})
	case r.URL.Path == "/UserViews":
		views := m.jellyfinLibraries()
		writeJSON(w, http.StatusOK, map[string]any{"Items": views, "TotalRecordCount": len(views)})
	case strings.HasPrefix(r.URL.Path, "/Shows/"):
		m.jellyfinShows(w, r)
	case r.URL.Path == "/Items" || strings.HasSuffix(r.URL.Path, "/Items"):
		m.jellyfinItems(w, r)
	case strings.HasPrefix(r.URL.Path, "/Users/") && strings.Contains(r.URL.Path, "/Items/"):
		clone := r.Clone(r.Context())
		copied := *r.URL
		copied.Path = "/Items/" + path.Base(r.URL.Path)
		clone.URL = &copied
		m.jellyfinItemRoute(w, clone)
	case strings.HasPrefix(r.URL.Path, "/Items/"):
		m.jellyfinItemRoute(w, r)
	case strings.HasPrefix(r.URL.Path, "/Videos/"):
		m.jellyfinVideoStream(w, r)
	case strings.HasPrefix(r.URL.Path, "/Audio/"):
		m.jellyfinAudioStream(w, r)
	case strings.HasPrefix(r.URL.Path, "/Sessions/Playing"):
		m.jellyfinProgress(w, r)
	case r.URL.Path == "/Sessions/Capabilities/Full" && r.Method == http.MethodPost:
		w.WriteHeader(http.StatusNoContent)
	case r.URL.Path == "/Sessions" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, []any{})
	case strings.HasPrefix(r.URL.Path, "/DisplayPreferences/"):
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"Id": path.Base(r.URL.Path), "SortBy": "SortName", "RememberIndexing": false, "CustomPrefs": map[string]string{}})
		} else if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "This Jellyfin compatibility route is not implemented", http.StatusNotImplemented)
	}
}

func queryInt(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}
func (m *multiServer) jellyfinPlaybackInfo(w http.ResponseWriter, r *http.Request, id string) {
	library, item, ok := m.jellyfinVideoByID(id)
	if !ok {
		audioLibrary, audioItem, audioOK := m.jellyfinAudioByID(id)
		if !audioOK {
			http.NotFound(w, r)
			return
		}
		source := map[string]any{"Id": jellyfin.ID("source", audioItem.ID), "Path": "/Audio/" + id + "/stream?Static=true", "DirectStreamUrl": "/Audio/" + id + "/stream?Static=true", "Protocol": "Http", "Container": audioItem.Container, "Size": audioItem.Size, "Name": audioItem.Title, "IsRemote": false, "SupportsDirectPlay": true, "SupportsDirectStream": true, "SupportsTranscoding": false, "RunTimeTicks": int64(audioItem.Duration * 10_000_000), "MediaStreams": []map[string]any{{"Type": "Audio", "Index": 0}}}
		_ = audioLibrary
		writeJSON(w, http.StatusOK, map[string]any{"MediaSources": []any{source}, "PlaySessionId": jellyfin.ID("play", audioItem.ID+strconv.FormatInt(time.Now().Unix(), 10))})
		return
	}
	streams := []map[string]any{}
	for _, stream := range item.Video {
		streams = append(streams, map[string]any{"Codec": stream.Codec, "Type": "Video", "Index": stream.Index, "Width": stream.Width, "Height": stream.Height})
	}
	for _, stream := range item.Audio {
		streams = append(streams, map[string]any{"Codec": stream.Codec, "Type": "Audio", "Index": stream.Index, "Language": stream.Language, "Title": stream.Title, "Channels": stream.Channels, "IsDefault": stream.Default})
	}
	source := map[string]any{"Id": jellyfin.ID("source", item.ID), "Path": "/Videos/" + id + "/stream?Static=true", "DirectStreamUrl": "/Videos/" + id + "/stream?Static=true", "Protocol": "Http", "Container": item.Container, "Size": item.Size, "Name": item.Title, "IsRemote": false, "SupportsDirectPlay": item.DirectPlay, "SupportsDirectStream": item.DirectPlay, "SupportsTranscoding": false, "RunTimeTicks": int64(item.Duration * 10_000_000), "MediaStreams": streams}
	_ = library
	writeJSON(w, http.StatusOK, map[string]any{"MediaSources": []any{source}, "PlaySessionId": jellyfin.ID("play", item.ID+strconv.FormatInt(time.Now().Unix(), 10))})
}
func (m *multiServer) jellyfinVideoStream(w http.ResponseWriter, r *http.Request) {
	id := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/Videos/"), "/"), "/")[0]
	library, item, ok := m.jellyfinVideoByID(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, _, err := library.Video.Open(item.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", item.MediaType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, path.Base(item.Path), item.Modified, file)
}
func (m *multiServer) jellyfinAudioStream(w http.ResponseWriter, r *http.Request) {
	id := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/Audio/"), "/"), "/")[0]
	library, item, ok := m.jellyfinAudioByID(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	file, _, err := library.Audio.Open(item.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", item.MediaType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, path.Base(item.Path), item.Modified, file)
}
func (m *multiServer) jellyfinProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var value struct {
		ItemID        string `json:"ItemId"`
		PositionTicks int64  `json:"PositionTicks"`
		IsPaused      bool   `json:"IsPaused"`
		PlayMethod    string `json:"PlayMethod"`
	}
	if decodeJSONRequest(w, r, 64<<10, &value, false) != nil {
		http.Error(w, "Invalid progress", http.StatusBadRequest)
		return
	}
	library, item, ok := m.jellyfinVideoByID(value.ItemID)
	if !ok {
		audioLibrary, audioItem, audioOK := m.jellyfinAudioByID(value.ItemID)
		if !audioOK {
			http.NotFound(w, r)
			return
		}
		if m.audioProgress != nil {
			account := m.readingAccount(r)
			position := float64(value.PositionTicks) / 10_000_000
			if position < 0 {
				position = 0
			}
			// A catalogue duration of zero means nothing probed the file; it is not the
			// length. Writing it over the duration the browser already learned loses the
			// known length, and the store then rejects a position past five seconds as
			// unplaceable, so the device's report vanished. Keep what is known instead.
			duration := audioItem.Duration
			if duration <= 0 {
				if saved, ok := m.audioProgress.Get(account, audioLibrary.ID, audioItem.ID); ok {
					duration = saved.Duration
				}
			}
			if duration > 0 && position > duration {
				position = duration
			}
			_ = m.audioProgress.Put(account, audio.Progress{LibraryID: audioLibrary.ID, ItemID: audioItem.ID, Position: position, Duration: duration, PlaybackRate: 1})
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if m.videoProgress != nil {
		account := m.readingAccount(r)
		position := float64(value.PositionTicks) / 10_000_000
		if position < 0 {
			position = 0
		}
		// Same rule as the audio branch: a zero catalogue duration is a missing probe,
		// not the file's length, and overwriting the known one drops the report.
		duration := item.Duration
		if duration <= 0 {
			if saved, ok := m.videoProgress.Get(account, library.ID, item.ID); ok {
				duration = saved.Duration
			}
		}
		if duration > 0 && position > duration {
			position = duration
		}
		_ = m.videoProgress.Put(account, video.Progress{LibraryID: library.ID, ItemID: item.ID, Position: position, Duration: duration, DeviceID: "jellyfin"})
	}
	w.WriteHeader(http.StatusNoContent)
}
