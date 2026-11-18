package appconfig

import "fmt"

// Module is an optional part of Mochil. The core (books, comics, manga,
// reading position, sync, accounts) is always on; a module that is off is never
// constructed: no routes, workers, caches or scans.
type Module struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Kinds       []string `json:"-"`
	// DefaultOn keeps a module running when nothing is set and no library
	// needs it, for parts that existing installs already rely on.
	DefaultOn bool `json:"default_on"`
	// Group names the card this module is a switch on, so related modules share
	// one card instead of showing as peers. Empty means the module is a card.
	Group string `json:"group,omitempty"`
	// Option marks a module as an extra switch inside its Group's card. It keeps
	// its own modules.<id> key and stays inert while the group's card is off.
	Option bool `json:"option,omitempty"`
}

// Modules lists every optional module in display order.
var Modules = []Module{
	{ID: "video", Name: "Video", Description: "Movies, TV and anime with streaming, subtitles and resume.", Kinds: []string{"movies", "tv", "anime"}},
	{ID: "audio", Name: "Audio", Description: "Audiobooks, podcasts and music with queue, playlists and chapters.", Kinds: []string{"audiobooks", "podcasts", "music"}},
	{ID: "documents", Name: "Documents", Description: "Documents and the wiki vault.", Kinds: []string{"documents", "vault"}},
	{ID: "narration", Name: "Narration", Description: "Read any book aloud with a local Pocket TTS voice. Needs the voice runtime installed.", Kinds: nil},
	{ID: "hardcover", Name: "Hardcover shelf sync", Description: "Mark books as currently reading or read on your Hardcover account. Uses the Hardcover token from the Metadata settings.", Kinds: nil},
	{ID: "hardcover_progress", Name: "Hardcover reading progress", Description: "Also send how far you are through each book to your Hardcover shelf, as a page count, at most every 15 minutes per book. Needs Hardcover shelf sync, and only touches books already on your shelf that Hardcover knows the page count of.", Kinds: nil, Group: "hardcover", Option: true},
	{ID: "hardcover_ratings", Name: "Hardcover ratings", Description: "Send the star rating you give a book in the Library editor to your Hardcover shelf. Needs Hardcover shelf sync, and only changes books already on your shelf that Mochil has matched.", Kinds: nil, Group: "hardcover", Option: true},
	{ID: "hermes", Name: "Hermes chat", Description: "Chat with your Hermes Agent from Mochil. Needs the Hermes API server address and key in the config (hermes.url, hermes.key).", Kinds: nil},
	{ID: "avatar", Name: "Avatar", Description: "A Live2D character beside the Hermes chat that blinks, moves and speaks along with her voice. Put model folders in the avatar folder and choose one in Settings, Avatar. Needs the Hermes module.", Kinds: nil},
	{ID: "editor", Name: "Library editor", Description: "A spreadsheet-style page to review and edit book titles, authors, series and numbers, one book or many at once. Edits are kept by Mochil and never change your files.", Kinds: nil},
	{ID: "android", Name: "Android app", Description: "Pairing, APK download and the API the Android app talks to.", DefaultOn: true},
}

// ModuleForKind returns the module a library kind belongs to, or "" for the core.
func ModuleForKind(kind string) string {
	for _, module := range Modules {
		for _, candidate := range module.Kinds {
			if candidate == kind {
				return module.ID
			}
		}
	}
	return ""
}

// ModuleEnabled reports whether a module runs. An explicit setting always wins;
// otherwise a module runs exactly when a library needs it, so an install that
// only has books never pays for video or audio.
func (cfg Config) ModuleEnabled(id string) bool {
	if enabled, ok := cfg.Modules[id]; ok {
		return enabled
	}
	for _, module := range Modules {
		if module.ID == id && module.DefaultOn {
			return true
		}
	}
	for _, library := range cfg.Libraries {
		if ModuleForKind(library.Kind) == id {
			return true
		}
	}
	return false
}

// EnabledModules returns the effective state of every module.
func (cfg Config) EnabledModules() map[string]bool {
	state := make(map[string]bool, len(Modules))
	for _, module := range Modules {
		state[module.ID] = cfg.ModuleEnabled(module.ID)
	}
	return state
}

func (cfg Config) validateModules() error {
	for id := range cfg.Modules {
		known := false
		for _, module := range Modules {
			known = known || module.ID == id
		}
		if !known {
			return fmt.Errorf("modules.%s is not a known module", id)
		}
	}
	return nil
}
