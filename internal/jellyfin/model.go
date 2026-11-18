// Package jellyfin contains the compatibility DTO boundary. It has no storage
// or HTTP dependencies and does not maintain a second media catalogue.
package jellyfin

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func ID(kind, value string) string {
	sum := sha256.Sum256([]byte("bobobooks-jellyfin\x00" + kind + "\x00" + value))
	return hex.EncodeToString(sum[:16])
}

type User struct {
	Name, ServerID, ID string
	HasPassword        bool
	Policy             map[string]any `json:"Policy"`
}

func NewUser(name, serverID string, hasPassword bool) map[string]any {
	return map[string]any{"Name": name, "ServerId": serverID, "Id": ID("user", name), "HasPassword": hasPassword, "HasConfiguredPassword": hasPassword, "Policy": map[string]any{"IsAdministrator": true, "EnableMediaPlayback": true, "EnableVideoPlaybackTranscoding": true, "EnableAudioPlaybackTranscoding": true}}
}
func Library(id, name, kind, server string) map[string]any {
	collection := "movies"
	itemType := "CollectionFolder"
	switch kind {
	case "tv", "anime":
		collection = "tvshows"
	case "music":
		collection = "music"
	case "audiobooks":
		collection = "books"
	}
	return map[string]any{"Name": name, "ServerId": server, "Id": ID("library", id), "Type": itemType, "CollectionType": collection, "IsFolder": true, "LocationType": "FileSystem", "MediaType": "Unknown"}
}
func Video(id, title, show string, season, episode int, duration float64, server string, isFolder bool) map[string]any {
	kind := "Movie"
	if show != "" {
		kind = "Episode"
	}
	value := map[string]any{"Name": title, "ServerId": server, "Id": ID("video", id), "Type": kind, "MediaType": "Video", "IsFolder": isFolder, "RunTimeTicks": int64(duration * 10_000_000), "LocationType": "FileSystem", "CanDownload": true, "PlayAccess": "Full"}
	if show != "" {
		value["SeriesName"] = show
		value["ParentIndexNumber"] = season
		value["IndexNumber"] = episode
	}
	return value
}
func Series(id, name, server string) map[string]any {
	return map[string]any{"Name": name, "ServerId": server, "Id": ID("series", id), "Type": "Series", "MediaType": "Video", "IsFolder": true, "LocationType": "FileSystem"}
}
func Season(id, name, seriesID, seriesName string, index int, server string) map[string]any {
	return map[string]any{"Name": name, "ServerId": server, "Id": ID("season", id), "SeriesId": ID("series", seriesID), "SeriesName": seriesName, "Type": "Season", "MediaType": "Video", "IsFolder": true, "IndexNumber": index, "LocationType": "FileSystem"}
}
func Audio(id, title, artist, album string, duration float64, server string) map[string]any {
	return map[string]any{"Name": title, "ServerId": server, "Id": ID("audio", id), "Type": "Audio", "MediaType": "Audio", "Artists": nonempty(artist), "Album": album, "RunTimeTicks": int64(duration * 10_000_000), "LocationType": "FileSystem", "CanDownload": true, "PlayAccess": "Full"}
}
func nonempty(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return []string{value}
}
