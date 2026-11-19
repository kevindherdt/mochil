package web

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"mochil/internal/appconfig"
)

// The avatar module draws a Live2D character beside the Hermes chat. Models are folders you place in the avatar
// directory (each holds a *.model3.json). The Live2D Cubism core, which has its own licence, is also placed there, in
// _runtime/live2dcubismcore.min.js. Everything is served to administrators only.

const (
	defaultAvatarDirectory = "/avatars"
	avatarRuntimeCore      = "_runtime/live2dcubismcore.min.js"
	maxAvatarModels        = 60
)

type avatarFile struct {
	Name string `json:"name"`
	File string `json:"file"`
}

type avatarModel struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Model       string       `json:"model"` // path of the .model3.json, relative to the model's folder
	Expressions []avatarFile `json:"expressions"`
	Motions     []avatarFile `json:"motions"`
}

func (m *multiServer) avatarDirectory() string {
	directory := m.cfg.Avatar.Directory
	if m.cfg.ConfigJSON != nil {
		if data, err := m.cfg.ConfigJSON(); err == nil {
			var document struct {
				Avatar *appconfig.Avatar `json:"avatar"`
			}
			if json.Unmarshal(data, &document) == nil && document.Avatar != nil {
				directory = document.Avatar.Directory
			}
		}
	}
	if strings.TrimSpace(directory) == "" {
		return defaultAvatarDirectory
	}
	return directory
}

// avatarModels lists the model folders: any folder up to four levels down that holds a .model3.json.
func avatarModels(root string) []avatarModel {
	var models []avatarModel
	_ = filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		relative, _ := filepath.Rel(root, current)
		if entry.IsDir() {
			if relative != "." && (strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".") || strings.Count(relative, string(filepath.Separator)) >= 4) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".model3.json") || len(models) >= maxAvatarModels {
			return nil
		}
		directory := filepath.Dir(relative)
		model := avatarModel{ID: filepath.ToSlash(directory), Name: filepath.Base(directory), Model: entry.Name(), Expressions: []avatarFile{}, Motions: []avatarFile{}}
		if directory == "." {
			model.ID, model.Name = ".", strings.TrimSuffix(entry.Name(), ".model3.json")
		}
		files, _ := os.ReadDir(filepath.Dir(current))
		for _, file := range files {
			switch {
			case strings.HasSuffix(file.Name(), ".exp3.json"):
				model.Expressions = append(model.Expressions, avatarFile{Name: strings.TrimSuffix(file.Name(), ".exp3.json"), File: file.Name()})
			case strings.HasSuffix(file.Name(), ".motion3.json"):
				model.Motions = append(model.Motions, avatarFile{Name: strings.TrimSuffix(file.Name(), ".motion3.json"), File: file.Name()})
			}
		}
		models = append(models, model)
		return fs.SkipDir // one model per folder
	})
	sort.Slice(models, func(i, j int) bool { return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID) })
	return models
}

// avatarChoice is what is kept: the chosen model, and the narration voice each model speaks with (a model without
// one uses the default narration voice).
type avatarChoice struct {
	Model  string            `json:"model"`
	Voices map[string]string `json:"voices,omitempty"`
	// Scene is what is behind her and Frames is where each model sits (see avatar_look.go).
	Scene  string                 `json:"scene,omitempty"`
	Frames map[string]avatarFrame `json:"frames,omitempty"`
}

func (m *multiServer) avatarSaved() avatarChoice {
	var saved avatarChoice
	if file := m.cfg.AvatarState; file != "" {
		if data, err := os.ReadFile(file); err == nil {
			_ = json.Unmarshal(data, &saved)
		}
	}
	return saved
}

// avatarSelected is the model chosen in Settings; with no valid choice it is the first model found.
func (m *multiServer) avatarSelected(models []avatarModel) string {
	saved := m.avatarSaved()
	for _, model := range models {
		if model.ID == saved.Model {
			return saved.Model
		}
	}
	if len(models) > 0 {
		return models[0].ID
	}
	return ""
}

func (m *multiServer) apiAvatar(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/avatar"), "/")
	root := m.avatarDirectory()
	switch {
	case rest == "models" && r.Method == http.MethodGet:
		models := avatarModels(root)
		_, coreErr := os.Stat(filepath.Join(root, filepath.FromSlash(avatarRuntimeCore)))
		w.Header().Set("Cache-Control", "no-store")
		voices := map[string]string{}
		for id, voice := range m.avatarSaved().Voices {
			if _, _, ok := m.narrationVoiceRef(voice); ok {
				voices[id] = voice
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": models, "selected": m.avatarSelected(models), "core": coreErr == nil, "voices": voices,
			"scene": m.avatarSaved().Scene, "frames": m.avatarSaved().Frames, "image": m.avatarBackgroundVersion()})
	case rest == "select" && r.Method == http.MethodPut:
		m.avatarSelect(w, r, root)
	case rest == "look" && r.Method == http.MethodPut:
		m.avatarLook(w, r, root)
	case rest == "background":
		m.avatarBackground(w, r)
	case rest == "core" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		m.serveAvatarFile(w, r, root, avatarRuntimeCore)
	case strings.HasPrefix(rest, "files/") && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		m.serveAvatarFile(w, r, root, strings.TrimPrefix(rest, "files/"))
	default:
		http.NotFound(w, r)
	}
}

func (m *multiServer) avatarSelect(w http.ResponseWriter, r *http.Request, root string) {
	if m.cfg.AvatarState == "" {
		http.Error(w, "The avatar cannot be saved on this server.", http.StatusNotImplemented)
		return
	}
	var request struct {
		Model string  `json:"model"`
		Voice *string `json:"voice"` // set the voice of this model; an empty text clears it
	}
	if err := decodeJSONRequest(w, r, 1<<10, &request, true); err != nil {
		http.Error(w, "Choose an avatar.", http.StatusBadRequest)
		return
	}
	known := false
	for _, model := range avatarModels(root) {
		known = known || model.ID == request.Model
	}
	if !known {
		http.Error(w, "Unknown avatar.", http.StatusBadRequest)
		return
	}
	avatarChoiceLock.Lock()
	defer avatarChoiceLock.Unlock()
	choice := m.avatarSaved()
	choice.Model = request.Model
	if request.Voice != nil {
		if choice.Voices == nil {
			choice.Voices = map[string]string{}
		}
		if *request.Voice == "" {
			delete(choice.Voices, request.Model)
		} else if _, _, ok := m.narrationVoiceRef(*request.Voice); ok {
			choice.Voices[request.Model] = *request.Voice
		} else {
			http.Error(w, "Unknown voice.", http.StatusBadRequest)
			return
		}
	}
	if !m.saveAvatarChoice(w, choice) {
		return
	}
	writeJSON(w, http.StatusOK, choice)
}

var avatarTypes = map[string]string{
	".json": "application/json", ".moc3": "application/octet-stream", ".png": "image/png", ".jpg": "image/jpeg",
	".jpeg": "image/jpeg", ".webp": "image/webp", ".mp3": "audio/mpeg", ".wav": "audio/wav", ".js": "text/javascript; charset=utf-8",
}

// serveAvatarFile serves one file from inside the avatar directory. The name is cleaned, only model file types are
// served, and a symbolic link that leads outside the directory is refused.
func (m *multiServer) serveAvatarFile(w http.ResponseWriter, r *http.Request, root, name string) {
	clean := path.Clean("/" + name)[1:]
	contentType, allowed := avatarTypes[strings.ToLower(path.Ext(clean))]
	if clean == "" || !allowed || strings.Contains(clean, "..") {
		http.NotFound(w, r)
		return
	}
	// Only the Cubism core may come from the runtime folder; models never start with an underscore.
	if strings.HasPrefix(clean, "_") && clean != avatarRuntimeCore {
		http.NotFound(w, r)
		return
	}
	full, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(clean)))
	base, baseErr := filepath.EvalSymlinks(root)
	if err != nil || baseErr != nil || !strings.HasPrefix(full, base+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}
	if stat, err := os.Stat(full); err == nil && !stat.IsDir() && strings.EqualFold(path.Ext(clean), ".png") && m.cfg.AvatarState != "" {
		full = avatarTextureForPhones(filepath.Join(filepath.Dir(m.cfg.AvatarState), "avatar-textures"), full, stat.Size(), stat.ModTime().UnixNano())
	}
	file, err := os.Open(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
}
