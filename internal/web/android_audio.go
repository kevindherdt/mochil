package web

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/audio"
	"mochil/internal/catalog"
)

type androidAudioGroup struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Subtitle         string `json:"subtitle,omitempty"`
	Count            int    `json:"count"`
	ArtworkItemID    string `json:"artwork_item_id,omitempty"`
	ArtworkLibraryID string `json:"artwork_library_id,omitempty"`
}

func (m *multiServer) androidAudioGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Audio == nil || (library.Kind != "music" && library.Kind != "audiobooks") {
		http.Error(w, "Unknown audio library", http.StatusNotFound)
		return
	}
	groupType := r.URL.Query().Get("group_type")
	if !androidAudioGroupingAllowed(library.Kind, groupType) {
		http.Error(w, "Unknown audio grouping", http.StatusBadRequest)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	out := buildAudioGroups(library.Audio.Items(), library.ID, groupType, query)
	offset, limit := boundedAndroidAudioPage(r, len(out))
	end := min(len(out), offset+limit)
	w.Header().Set("X-Total-Count", strconv.Itoa(len(out)))
	if end < len(out) {
		w.Header().Set("X-Next-Offset", strconv.Itoa(end))
	}
	writeJSON(w, http.StatusOK, out[offset:end])
}

// Shared by grouped browsing and its A–Z offsets; identities and filters must agree.
func buildAudioGroups(items []audio.Item, libraryID, groupType, query string) []androidAudioGroup {
	groups := make(map[string]*androidAudioGroup)
	matching := make(map[string]bool)
	for _, item := range items {
		name, subtitle := androidAudioGroupIdentity(item, groupType)
		if name == "" {
			continue
		}
		key := audioGroupItemID(libraryID, groupType, item)
		if query != "" && strings.Contains(strings.ToLower(name+" "+subtitle), query) {
			matching[key] = true
		}
		group := groups[key]
		if group == nil {
			group = &androidAudioGroup{ID: key, Name: name, Subtitle: subtitle, ArtworkItemID: item.ID, ArtworkLibraryID: libraryID}
			groups[key] = group
		} else if groupType == "albums" && !strings.EqualFold(group.Subtitle, subtitle) {
			group.Subtitle = "Various artists"
		}
		group.Count++
	}
	out := make([]androidAudioGroup, 0, len(groups))
	for key, group := range groups {
		// Search selects complete groups, never a subset of their tracks. Keep
		// contributor matches as well as the final displayed compilation label.
		if query != "" && !matching[key] && !strings.Contains(strings.ToLower(group.Name+" "+group.Subtitle), query) {
			continue
		}
		out = append(out, *group)
	}
	sortAudioGroups(out)
	return out
}

func (m *multiServer) androidAudioGroupTracks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok || library.Audio == nil || (library.Kind != "music" && library.Kind != "audiobooks") {
		http.Error(w, "Unknown audio library", http.StatusNotFound)
		return
	}
	groupType, wanted := r.URL.Query().Get("group_type"), strings.TrimSpace(r.URL.Query().Get("group"))
	wantedID := strings.TrimSpace(r.URL.Query().Get("group_id"))
	wantedArtist := strings.TrimSpace(r.URL.Query().Get("artist"))
	if !androidAudioGroupingAllowed(library.Kind, groupType) || wanted == "" && wantedID == "" {
		http.Error(w, "Invalid audio group", http.StatusBadRequest)
		return
	}
	items := audioGroupItems(library, groupType, wanted, wantedArtist, wantedID)
	sortAudioGroupItems(items, groupType)
	out := make([]audioView, 0, len(items))
	for _, item := range items {
		out = append(out, m.audioView(m.readingAccount(r), library, item))
	}
	writeJSON(w, http.StatusOK, out)
}

func sortAudioGroupItems(items []audio.Item, groupType string) {
	sort.SliceStable(items, func(i, j int) bool {
		if groupType == "albums" || groupType == "series" {
			if items[i].SeriesIndex != items[j].SeriesIndex {
				if items[i].SeriesIndex == 0 {
					return false
				}
				if items[j].SeriesIndex == 0 {
					return true
				}
				return items[i].SeriesIndex < items[j].SeriesIndex
			}
			if items[i].Disc != items[j].Disc {
				return items[i].Disc < items[j].Disc
			}
			if items[i].Track != items[j].Track {
				return items[i].Track < items[j].Track
			}
		}
		return catalog.NaturalLess(items[i].Title, items[j].Title)
	})
}

func audioGroupItems(library Library, groupType, wanted, wantedArtist, wantedID string) []audio.Item {
	items := make([]audio.Item, 0)
	for _, item := range library.Audio.Items() {
		name, _ := androidAudioGroupIdentity(item, groupType)
		artist, _ := androidAudioGroupIdentity(item, "artists")
		identityMatches := wantedID != "" && audioGroupItemID(library.ID, groupType, item) == wantedID
		nameMatches := wantedID == "" && strings.EqualFold(strings.TrimSpace(name), wanted)
		if (identityMatches || nameMatches) && (wantedArtist == "" || strings.EqualFold(strings.TrimSpace(artist), wantedArtist)) {
			items = append(items, item)
		}
	}
	return items
}

func (m *multiServer) androidArtistAlbums(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	artist := strings.TrimSpace(r.URL.Query().Get("artist"))
	if !ok || library.Audio == nil || library.Kind != "music" || artist == "" {
		http.Error(w, "Invalid music artist", http.StatusBadRequest)
		return
	}
	out := androidArtistAlbumGroups(library.Audio.Items(), library.ID, artist)
	writeJSON(w, http.StatusOK, out)
}

func androidArtistAlbumGroups(items []audio.Item, libraryID, artist string) []androidAudioGroup {
	wanted := map[string]bool{}
	for _, item := range items {
		itemArtist, _ := androidAudioGroupIdentity(item, "artists")
		if strings.EqualFold(strings.TrimSpace(itemArtist), artist) {
			wanted[audioGroupItemID(libraryID, "albums", item)] = true
		}
	}
	groups := map[string]*androidAudioGroup{}
	for _, item := range items {
		album, _ := androidAudioGroupIdentity(item, "albums")
		key := audioGroupItemID(libraryID, "albums", item)
		if !wanted[key] || album == "" {
			continue
		}
		group := groups[key]
		if group == nil {
			_, subtitle := androidAudioGroupIdentity(item, "albums")
			group = &androidAudioGroup{ID: key, Name: album, Subtitle: subtitle, ArtworkItemID: item.ID, ArtworkLibraryID: libraryID}
			groups[key] = group
		} else if _, subtitle := androidAudioGroupIdentity(item, "albums"); !strings.EqualFold(group.Subtitle, subtitle) {
			group.Subtitle = "Various artists"
		}
		group.Count++
	}
	out := make([]androidAudioGroup, 0, len(groups))
	for _, group := range groups {
		out = append(out, *group)
	}
	sortAudioGroups(out)
	return out
}

func androidAudioGroupIdentity(item audio.Item, groupType string) (string, string) {
	if groupType == "series" {
		return audiobookGroupName(item), strings.TrimSpace(item.Artist)
	}
	if groupType == "authors" {
		return strings.TrimSpace(item.Artist), "Author"
	}
	if groupType == "artists" {
		name := strings.TrimSpace(item.AlbumArtist)
		if name == "" {
			name = strings.TrimSpace(item.Artist)
		}
		return name, "Artist"
	}
	artist := strings.TrimSpace(item.AlbumArtist)
	if artist == "" {
		artist = strings.TrimSpace(item.Artist)
	}
	return strings.TrimSpace(item.Album), artist
}

func audioGroupID(libraryID, groupType, name, subtitle string) string {
	identity := strings.ToLower(strings.TrimSpace(libraryID)) + "\x00" + strings.ToLower(strings.TrimSpace(groupType)) + "\x00" + strings.ToLower(strings.TrimSpace(name)) + "\x00" + strings.ToLower(strings.TrimSpace(subtitle))
	digest := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(digest[:16])
}

func audioGroupItemID(libraryID, groupType string, item audio.Item) string {
	name, subtitle := androidAudioGroupIdentity(item, groupType)
	if groupType == "albums" {
		folder := path.Dir(strings.TrimSpace(item.Path))
		if audio.IsDiscFolder(path.Base(folder)) {
			folder = path.Dir(folder)
		}
		if folder != "" && folder != "." {
			subtitle = folder
		}
	}
	return audioGroupID(libraryID, groupType, name, subtitle)
}

func androidAudioGroupingAllowed(kind, groupType string) bool {
	if kind == "music" {
		return groupType == "albums" || groupType == "artists"
	}
	return kind == "audiobooks" && (groupType == "series" || groupType == "authors")
}

// boundedAndroidAudioPage is the page window the grouped audio shelf and the
// podcast show list share: same default, same ceiling, same end-of-list rule.
func boundedAndroidAudioPage(r *http.Request, total int) (int, int) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if offset < 0 {
		offset = 0
	}
	// A rescan can shrink the catalogue between page requests. End the page
	// sequence instead of replaying its first page into the client's list.
	if offset > total {
		offset = total
	}
	if limit < 1 || limit > 120 {
		limit = 60
	}
	return offset, limit
}
