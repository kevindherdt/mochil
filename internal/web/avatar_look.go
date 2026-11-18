package web

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// How the avatar looks: the scene behind her, and for each model how big it sits in the frame and where. The scenes are
// drawn by the page (gradients), except "image", which is one picture the administrator uploads.

const maxAvatarBackgroundBytes = 6 << 20

var avatarScenes = map[string]bool{"": true, "clear": true, "room": true, "night": true, "sunset": true, "studio": true, "mint": true, "rain": true, "desk": true, "loft": true, "balcony": true, "image": true}

// avatarFrame is the size and place of a model in its frame: zoom 1 is the default, x and y move it by up to half the frame.
type avatarFrame struct {
	Zoom float64 `json:"zoom"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

func (f avatarFrame) valid() bool {
	return f.Zoom >= 0.3 && f.Zoom <= 2.2 && f.X >= -1 && f.X <= 1 && f.Y >= -1 && f.Y <= 1
}

func (m *multiServer) avatarBackgroundFile() string {
	if m.cfg.AvatarState == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(m.cfg.AvatarState), "avatar-background")
}

// avatarBackgroundVersion is the saved picture's change time, or 0 when there is none; the page uses it to refresh.
func (m *multiServer) avatarBackgroundVersion() int64 {
	file := m.avatarBackgroundFile()
	if file == "" {
		return 0
	}
	if info, err := os.Stat(file); err == nil {
		return info.ModTime().UnixNano() / 1e6
	}
	return 0
}

// avatarChoiceLock serialises changes to the saved avatar choice.
var avatarChoiceLock sync.Mutex

func (m *multiServer) saveAvatarChoice(w http.ResponseWriter, choice avatarChoice) bool {
	data, _ := json.Marshal(choice)
	if err := os.MkdirAll(filepath.Dir(m.cfg.AvatarState), 0o755); err != nil {
		http.Error(w, "Could not save the avatar.", http.StatusInternalServerError)
		return false
	}
	temporary := m.cfg.AvatarState + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil || os.Rename(temporary, m.cfg.AvatarState) != nil {
		http.Error(w, "Could not save the avatar.", http.StatusInternalServerError)
		return false
	}
	return true
}

// avatarLook saves the scene and/or the frame of one model.
func (m *multiServer) avatarLook(w http.ResponseWriter, r *http.Request, root string) {
	if m.cfg.AvatarState == "" {
		http.Error(w, "The avatar cannot be saved on this server.", http.StatusNotImplemented)
		return
	}
	var request struct {
		Scene *string `json:"scene"`
		Frame *struct {
			Model string `json:"model"`
			avatarFrame
		} `json:"frame"`
	}
	if err := decodeJSONRequest(w, r, 1<<10, &request, true); err != nil || (request.Scene == nil && request.Frame == nil) {
		http.Error(w, "Choose a scene or a frame.", http.StatusBadRequest)
		return
	}
	avatarChoiceLock.Lock() // read, change and write as one step, so two devices cannot overwrite each other
	defer avatarChoiceLock.Unlock()
	choice := m.avatarSaved()
	if request.Scene != nil {
		if !avatarScenes[*request.Scene] || (*request.Scene == "image" && m.avatarBackgroundVersion() == 0) {
			http.Error(w, "Unknown scene.", http.StatusBadRequest)
			return
		}
		choice.Scene = *request.Scene
	}
	if request.Frame != nil {
		known := false
		for _, model := range avatarModels(root) {
			known = known || model.ID == request.Frame.Model
		}
		if !known || !request.Frame.avatarFrame.valid() {
			http.Error(w, "That frame is not allowed.", http.StatusBadRequest)
			return
		}
		if choice.Frames == nil {
			choice.Frames = map[string]avatarFrame{}
		}
		if request.Frame.avatarFrame == (avatarFrame{Zoom: 1}) {
			delete(choice.Frames, request.Frame.Model)
		} else {
			choice.Frames[request.Frame.Model] = request.Frame.avatarFrame
		}
	}
	if !m.saveAvatarChoice(w, choice) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scene": choice.Scene, "frames": choice.Frames})
}

// avatarBackground stores, serves or removes the one uploaded picture.
func (m *multiServer) avatarBackground(w http.ResponseWriter, r *http.Request) {
	file := m.avatarBackgroundFile()
	if file == "" {
		http.Error(w, "The avatar cannot be saved on this server.", http.StatusNotImplemented)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := os.ReadFile(file)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", http.DetectContentType(data))
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	case http.MethodPut:
		limitRequestBody(w, r, maxAvatarBackgroundBytes+1, uploadBodyBudget)
		data, err := io.ReadAll(r.Body)
		if err != nil || len(data) == 0 || len(data) > maxAvatarBackgroundBytes {
			http.Error(w, "That picture is too large (6 MB at most).", http.StatusRequestEntityTooLarge)
			return
		}
		switch kind := http.DetectContentType(data); kind {
		case "image/jpeg", "image/png", "image/webp":
		default:
			http.Error(w, "Use a JPEG, PNG or WebP picture.", http.StatusUnsupportedMediaType)
			return
		}
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			http.Error(w, "Could not save the picture.", http.StatusInternalServerError)
			return
		}
		temporary := file + ".tmp"
		if err := os.WriteFile(temporary, data, 0o644); err != nil || os.Rename(temporary, file) != nil {
			http.Error(w, "Could not save the picture.", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"version": m.avatarBackgroundVersion()})
	case http.MethodDelete:
		avatarChoiceLock.Lock()
		defer avatarChoiceLock.Unlock()
		_ = os.Remove(file)
		if choice := m.avatarSaved(); strings.EqualFold(choice.Scene, "image") {
			choice.Scene = ""
			if !m.saveAvatarChoice(w, choice) {
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
