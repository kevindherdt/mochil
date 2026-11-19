package web

import (
	"archive/zip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"

	"mochil/internal/audio"
)

const (
	maximumAudioArchiveItems = 1000
	maximumAudioArchiveBytes = int64(50 << 30)
)

var audioArchiveSlots = make(chan struct{}, 2)

type audioArchiveItem struct {
	library Library
	item    audio.Item
}

func safeArchiveName(value string) string {
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || strings.ContainsRune(`/\\:*?"<>|`, character) {
			return ' '
		}
		return character
	}, value)
	value = strings.Trim(strings.Join(strings.Fields(value), " "), ". ")
	if value == "" {
		return "Playlist"
	}
	return value
}

func validateAudioArchiveItems(items []audioArchiveItem, subject string) error {
	if len(items) == 0 {
		return fmt.Errorf("%s has no available tracks", subject)
	}
	if len(items) > maximumAudioArchiveItems {
		return fmt.Errorf("%s exceeds the %d-track download limit", subject, maximumAudioArchiveItems)
	}
	var total int64
	for _, entry := range items {
		if entry.item.Size <= 0 || total > maximumAudioArchiveBytes-entry.item.Size {
			return fmt.Errorf("%s exceeds the 50 GB download limit", subject)
		}
		total += entry.item.Size
	}
	return nil
}

func (m *multiServer) playlistDownloadItems(account, name string) ([]audioArchiveItem, bool, error) {
	playlist, found := m.audioPlaylists.Get(account, name)
	if !found {
		return nil, false, nil
	}
	entries := playlist.Entries
	if len(entries) == 0 {
		return nil, true, fmt.Errorf("playlist is empty")
	}
	items := make([]audioArchiveItem, 0, len(entries))
	for _, entry := range entries {
		library, ok := m.library(entry.LibraryID)
		if !ok || library.Audio == nil || library.Kind != "music" {
			continue
		}
		item, ok := library.Audio.Item(entry.ItemID)
		if !ok {
			continue
		}
		items = append(items, audioArchiveItem{library: library, item: item})
	}
	if err := validateAudioArchiveItems(items, "playlist"); err != nil {
		return nil, true, err
	}
	return items, true, nil
}

func (m *multiServer) downloadAudioPlaylist(w http.ResponseWriter, r *http.Request, account, name string) {
	items, found, err := m.playlistDownloadItems(account, name)
	if !found {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	streamAudioArchive(w, r, name, items)
}

func (m *multiServer) audioGroupDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	groupID := strings.TrimSpace(r.URL.Query().Get("group_id"))
	if !ok || library.Audio == nil || library.Kind != "music" || len(groupID) != 32 {
		http.NotFound(w, r)
		return
	}
	groupItems := audioGroupItems(library, "albums", "", "", groupID)
	if len(groupItems) == 0 {
		http.NotFound(w, r)
		return
	}
	sortAudioGroupItems(groupItems, "albums")
	items := make([]audioArchiveItem, 0, len(groupItems))
	for _, item := range groupItems {
		items = append(items, audioArchiveItem{library: library, item: item})
	}
	if err := validateAudioArchiveItems(items, "album"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name, _ := androidAudioGroupIdentity(groupItems[0], "albums")
	streamAudioArchive(w, r, name, items)
}

func streamAudioArchive(w http.ResponseWriter, r *http.Request, name string, items []audioArchiveItem) {
	select {
	case audioArchiveSlots <- struct{}{}:
		defer func() { <-audioArchiveSlots }()
	default:
		http.Error(w, "Two collection downloads are already running", http.StatusTooManyRequests)
		return
	}

	archiveName := safeArchiveName(name)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": archiveName + ".zip"}))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	writer := zip.NewWriter(w)
	var err error
	width := len(fmt.Sprint(len(items)))
	for index, entry := range items {
		if err = r.Context().Err(); err != nil {
			break
		}
		file, opened, openErr := entry.library.Audio.Open(entry.item.ID)
		if openErr != nil {
			err = openErr
			break
		}
		info, statErr := file.Stat()
		if statErr != nil {
			file.Close()
			err = statErr
			break
		}
		header, headerErr := zip.FileInfoHeader(info)
		if headerErr != nil {
			file.Close()
			err = headerErr
			break
		}
		header.Name = archiveName + "/" + fmt.Sprintf("%0*d - %s", width, index+1, safeArchiveName(filepath.Base(opened.Path)))
		header.Method = zip.Store
		header.SetMode(0o644)
		destination, createErr := writer.CreateHeader(header)
		if createErr == nil {
			_, createErr = io.Copy(destination, file)
		}
		file.Close()
		if createErr != nil {
			err = createErr
			break
		}
	}
	_ = writer.Close()
	if err != nil {
		// The response has already begun as a 200 ZIP, so the only honest signal left
		// is to abort the transfer rather than hand back a truncated archive.
		panic(http.ErrAbortHandler)
	}
}
