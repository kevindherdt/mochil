package web

import "mochil/internal/audio"

// Keep collection and detail responses consistent when library files disappear.
func (m *multiServer) playlistView(account string, list audio.Playlist) audioPlaylistView {
	view := audioPlaylistView{Name: list.Name, Items: make([]audioView, 0, len(list.Entries))}
	for _, entry := range list.Entries {
		library, ok := m.library(entry.LibraryID)
		if !ok || library.Audio == nil || library.Kind != "music" {
			continue
		}
		if item, ok := library.Audio.Item(entry.ItemID); ok {
			view.Items = append(view.Items, m.audioView(account, library, item))
		}
	}
	return view
}
