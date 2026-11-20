package web

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"time"
)

// koReaderPluginVersion changes whenever main.lua does, so devices can tell
// which build they run.
const koReaderPluginVersion = "1.28.0"

// main.lua is a small loader that rarely changes; the plugin itself is the core, which a paired device can
// replace from inside KOReader (see koReaderPluginManifest).
//
//go:embed koreader_plugin/main.lua
var koReaderLoaderLua string

//go:embed koreader_plugin/core.lua
var koReaderCoreLua string

// koReaderLoaderAPI is the loader interface the core expects. A core that needs a newer loader cannot be
// installed by an update; the reader is told to download the plugin again.
const koReaderLoaderAPI = 1

// koReaderPluginManifest describes the core a device may download.
func koReaderPluginManifest() map[string]any {
	sum := sha256.Sum256([]byte(koReaderCoreLua))
	return map[string]any{
		"version":    koReaderPluginVersion,
		"sha256":     hex.EncodeToString(sum[:]),
		"size":       len(koReaderCoreLua),
		"loader_api": koReaderLoaderAPI,
	}
}

func buildKoReaderPlugin(serverURL, token string) ([]byte, error) {
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	files := []struct{ name, body string }{
		{"mochil.koplugin/_meta.lua", fmt.Sprintf(`local _ = require("gettext")
return { fullname = _("Mochil Sync"), description = _("Accurate reading-position and highlight sync between KOReader and Mochil."), version = %q }
`, koReaderPluginVersion)},
		{"mochil.koplugin/mochil_config.lua", fmt.Sprintf("return { server_url = %q, token = %q }\n", serverURL, token)},
		{"mochil.koplugin/main.lua", koReaderLoaderLua},
		{"mochil.koplugin/mochil_core.lua", koReaderCoreLua},
		{"mochil.koplugin/README.txt", "Mochil Sync\n\nExtract the mochil.koplugin folder into koreader/plugins/ and restart KOReader. The plugin is already paired with your server and synchronizes reading positions and highlights. You can generate and install separate packages on your Kobo, Boox, phone, and tablet; existing devices remain paired.\n\nUpdates: open Tools, then Mochil Sync, then Check for plugin update. The plugin updates itself and keeps your pairing, so you only need to install it by hand once.\n\nAddress: this package points at the address it was downloaded from. If that address stops being the right one -- the server moved to a domain, a new port or a new IP -- open Tools, then Mochil Sync, tap the Server line and type the new address. Nothing needs reinstalling, and emptying the field returns to the address above.\n"},
	}
	for _, file := range files {
		f, err := zw.CreateHeader(&zip.FileHeader{Name: file.name, Method: zip.Deflate, Modified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			return nil, err
		}
		if _, err = f.Write([]byte(file.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
