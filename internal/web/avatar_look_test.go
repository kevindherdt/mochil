package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func avatarCall(m *multiServer, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	m.apiAvatar(w, request)
	return w
}

func tinyPNG() []byte {
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	picture.Set(0, 0, color.RGBA{R: 255, A: 255})
	var out bytes.Buffer
	_ = png.Encode(&out, picture)
	return out.Bytes()
}

func TestAvatarSceneAndFrameAreSavedAndKeptWhenTheModelChanges(t *testing.T) {
	m, _ := avatarFixture(t)
	model := "Ice Girl/Pack"
	if got := avatarCall(m, http.MethodPut, "/api/avatar/look", "application/json", []byte(`{"scene":"night","frame":{"model":"`+model+`","zoom":1.4,"x":0.1,"y":-0.2}}`)).Code; got != http.StatusOK {
		t.Fatalf("look: %d", got)
	}
	if got := avatarCall(m, http.MethodPut, "/api/avatar/select", "application/json", []byte(`{"model":"`+model+`"}`)).Code; got != http.StatusOK {
		t.Fatalf("select: %d", got)
	}
	var listed struct {
		Scene  string                 `json:"scene"`
		Frames map[string]avatarFrame `json:"frames"`
		Image  int64                  `json:"image"`
	}
	_ = json.Unmarshal(avatarCall(m, http.MethodGet, "/api/avatar/models", "", nil).Body.Bytes(), &listed)
	if listed.Scene != "night" || listed.Frames[model].Zoom != 1.4 || listed.Frames[model].Y != -0.2 || listed.Image != 0 {
		t.Fatalf("saved look: %+v", listed)
	}
	// The default frame is not stored, so a reset leaves nothing behind.
	avatarCall(m, http.MethodPut, "/api/avatar/look", "application/json", []byte(`{"frame":{"model":"`+model+`","zoom":1,"x":0,"y":0}}`))
	_ = json.Unmarshal(avatarCall(m, http.MethodGet, "/api/avatar/models", "", nil).Body.Bytes(), &listed)
	if len(listed.Frames) != 0 || listed.Scene != "night" {
		t.Fatalf("reset: %+v", listed)
	}
}

func TestAvatarLookRefusesWhatIsNotAllowed(t *testing.T) {
	m, _ := avatarFixture(t)
	for name, body := range map[string]string{
		"unknown scene":         `{"scene":"volcano"}`,
		"image without picture": `{"scene":"image"}`,
		"too much zoom":         `{"frame":{"model":"Other","zoom":9,"x":0,"y":0}}`,
		"off the frame":         `{"frame":{"model":"Other","zoom":1,"x":3,"y":0}}`,
		"unknown model":         `{"frame":{"model":"Nope","zoom":1,"x":0,"y":0}}`,
		"nothing":               `{}`,
	} {
		if got := avatarCall(m, http.MethodPut, "/api/avatar/look", "application/json", []byte(body)).Code; got != http.StatusBadRequest {
			t.Errorf("%s: %d", name, got)
		}
	}
}

func TestAvatarBackgroundPictureIsCheckedStoredServedAndRemoved(t *testing.T) {
	m, _ := avatarFixture(t)
	if got := avatarCall(m, http.MethodGet, "/api/avatar/background", "", nil).Code; got != http.StatusNotFound {
		t.Fatalf("none yet: %d", got)
	}
	if got := avatarCall(m, http.MethodPut, "/api/avatar/background", "image/png", []byte("<html>not a picture</html>")).Code; got != http.StatusUnsupportedMediaType {
		t.Fatalf("a page must be refused: %d", got)
	}
	if got := avatarCall(m, http.MethodPut, "/api/avatar/background", "image/png", append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte("x"), maxAvatarBackgroundBytes)...)).Code; got != http.StatusRequestEntityTooLarge {
		t.Fatalf("too large: %d", got)
	}
	if got := avatarCall(m, http.MethodPut, "/api/avatar/background", "image/png", tinyPNG()).Code; got != http.StatusOK {
		t.Fatalf("store: %d", got)
	}
	got := avatarCall(m, http.MethodGet, "/api/avatar/background", "", nil)
	if got.Code != http.StatusOK || got.Header().Get("Content-Type") != "image/png" || !bytes.Equal(got.Body.Bytes(), tinyPNG()) {
		t.Fatalf("serve: %d %s", got.Code, got.Header().Get("Content-Type"))
	}
	if code := avatarCall(m, http.MethodPut, "/api/avatar/look", "application/json", []byte(`{"scene":"image"}`)).Code; code != http.StatusOK {
		t.Fatalf("the picture can be the scene once it exists: %d", code)
	}
	if code := avatarCall(m, http.MethodDelete, "/api/avatar/background", "", nil).Code; code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	if code := avatarCall(m, http.MethodGet, "/api/avatar/background", "", nil).Code; code != http.StatusNotFound {
		t.Fatalf("gone: %d", code)
	}
	if !strings.Contains(avatarCall(m, http.MethodGet, "/api/avatar/models", "", nil).Body.String(), `"scene":""`) && strings.Contains(avatarCall(m, http.MethodGet, "/api/avatar/models", "", nil).Body.String(), `"scene":"image"`) {
		t.Fatal("removing the picture must also end the scene that showed it")
	}
}
