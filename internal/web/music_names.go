package web

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mochil/internal/audio"
	"mochil/internal/musicnames"
	"mochil/internal/musicrename"
)

// The File names tool for music albums (administrators). It renames files inside their own folder from their tags,
// optionally after a model has tidied messy tags. A track's id comes from its path, so after a rename the queue,
// favourites, playlists and progress are carried to the new id, and the library is rescanned. Every batch is recorded
// in a ledger and can be undone.

const maximumTidyTracks = 80

func (m *multiServer) hermesMusicNames(w http.ResponseWriter, r *http.Request) {
	switch strings.TrimPrefix(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/hermes"), "/"), "music-names") {
	case "":
		m.musicNamesPlan(w, r)
	case "/apply":
		m.musicNamesApply(w, r)
	case "/undo":
		m.musicNamesUndo(w, r)
	case "/last":
		m.musicNamesLast(w, r)
	default:
		http.NotFound(w, r)
	}
}

// musicAlbum finds the album's library and tracks, or answers 404.
func (m *multiServer) musicAlbum(w http.ResponseWriter, library, groupID string) (Library, []audio.Item, bool) {
	lib, ok := m.library(strings.TrimSpace(library))
	groupID = strings.TrimSpace(groupID)
	if !ok || lib.Audio == nil || lib.Kind != "music" || len(groupID) != 32 || lib.SourcePath == "" {
		http.NotFound(w, nil)
		return Library{}, nil, false
	}
	items := audioGroupItems(lib, "albums", "", "", groupID)
	if len(items) == 0 {
		http.NotFound(w, nil)
		return Library{}, nil, false
	}
	sortAudioGroupItems(items, "albums")
	return lib, items, true
}

func albumTracks(items []audio.Item) []musicnames.Track {
	tracks := make([]musicnames.Track, len(items))
	for i, item := range items {
		tracks[i] = musicnames.Track{ID: item.ID, Path: item.Path, Title: item.Title, Artist: item.Artist, AlbumArtist: item.AlbumArtist, Album: item.Album, Track: item.Track, Disc: item.Disc}
	}
	return tracks
}

func (m *multiServer) musicNamesPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	lib, items, ok := m.musicAlbum(w, r.URL.Query().Get("library"), r.URL.Query().Get("group_id"))
	if !ok {
		return
	}
	tracks, notes := albumTracks(items), make([][]string, len(items))
	tidied, modelNote := false, ""
	if r.URL.Query().Get("tidy") == "1" {
		// Fixed rules first: web addresses, "(Official Video)", bitrate noise, track-number prefixes, ALL CAPS.
		tracks, notes = musicnames.PreClean(tracks)
		tidied = true
		settings := m.hermesSettings()
		if settings.HelperConfigured() && settings.Feature("music") && len(tracks) <= maximumTidyTracks {
			ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
			reply, err := hermesAsk(ctx, settings, musicnames.TidyInstructions, musicnames.TidyMessage(tracks))
			cancel()
			if tidy, parsed := musicnames.ParseTidy(reply); err == nil && parsed {
				var more [][]string
				tracks, more = musicnames.ApplyTidy(tracks, tidy)
				for i := range notes {
					notes[i] = append(notes[i], more[i]...)
				}
			} else {
				modelNote = "The helper model gave no usable answer, so only the fixed cleaning rules were used."
			}
		}
	}
	taken := func(relative string) bool {
		_, err := os.Lstat(filepath.Join(lib.SourcePath, filepath.FromSlash(relative)))
		return err == nil
	}
	proposals := musicnames.ProposeFileNames(tracks, taken)
	type row struct {
		musicnames.Proposal
		Tidy []string `json:"tidy,omitempty"`
	}
	rows := make([]row, len(proposals))
	counts := map[string]int{}
	for i, proposal := range proposals {
		rows[i] = row{Proposal: proposal, Tidy: notes[i]}
		counts[proposal.Status]++
	}
	name, _ := androidAudioGroupIdentity(items[0], "albums")
	last, canUndo := m.musicEngine(lib).Latest(lib.ID)
	writeJSON(w, http.StatusOK, map[string]any{"album": name, "tracks": rows, "change": counts["change"], "same": counts["same"], "problem": counts["problem"], "tidied": tidied, "model_note": modelNote, "undo": canUndo && len(last.Moves) > 0})
}

// musicEngine is one engine per library, so two batches never run at once.
func (m *multiServer) musicEngine(lib Library) *musicrename.Engine {
	engine, _ := m.musicEngines.LoadOrStore(lib.ID, &musicrename.Engine{Root: lib.SourcePath, Dir: filepath.Join(filepath.Dir(m.cfg.AudioState), "music-renames")})
	return engine.(*musicrename.Engine)
}
