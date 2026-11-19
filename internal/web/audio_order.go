package web

import (
	"strings"

	"mochil/internal/audio"
	"mochil/internal/catalog"
)

func audioItemLess(a, b audio.Item, sortOrder, libraryKind string) bool {
	switch sortOrder {
	case "recent":
		if !a.Modified.Equal(b.Modified) {
			return a.Modified.After(b.Modified)
		}
	case "oldest":
		if !a.Modified.Equal(b.Modified) {
			return a.Modified.Before(b.Modified)
		}
	case "title":
		// The common title comparison below includes a stable identity tie-break.
	case "artist":
		ak, bk := strings.ToLower(a.Artist+"\x00"+a.Album), strings.ToLower(b.Artist+"\x00"+b.Album)
		if ak != bk {
			return ak < bk
		}
		if a.Disc != b.Disc {
			return a.Disc < b.Disc
		}
		if a.Track != b.Track {
			return a.Track < b.Track
		}
		if ak, bk := strings.ToLower(a.Title), strings.ToLower(b.Title); ak != bk {
			return ak < bk
		}
	case "album":
		ak, bk := catalog.SortKey(a.Album), catalog.SortKey(b.Album)
		if ak != bk {
			return ak < bk
		}
		if a.Disc != b.Disc {
			return a.Disc < b.Disc
		}
		if a.Track != b.Track {
			return a.Track < b.Track
		}
	case "series":
		as, bs := strings.TrimSpace(a.Series), strings.TrimSpace(b.Series)
		if as == "" {
			as = strings.TrimSpace(a.Album)
		}
		if bs == "" {
			bs = strings.TrimSpace(b.Album)
		}
		if order := compareAudioLabels(as, bs); order != 0 {
			return order < 0
		}
		ai, bi := a.SeriesIndex, b.SeriesIndex
		if ai == 0 {
			ai = float64(a.Track)
		}
		if bi == 0 {
			bi = float64(b.Track)
		}
		if ai != bi {
			if ai == 0 {
				return false
			}
			if bi == 0 {
				return true
			}
			return ai < bi
		}
		if a.Disc != b.Disc {
			return a.Disc < b.Disc
		}
		if a.Track != b.Track {
			return a.Track < b.Track
		}
	default:
		if libraryKind == "podcasts" && !a.Modified.Equal(b.Modified) {
			return a.Modified.After(b.Modified)
		}
	}
	if order := compareAudioLabels(a.Title, b.Title); order != 0 {
		return order < 0
	}
	return a.ID < b.ID
}

// Normalize once, then distinguish a true order from equivalent spelling/number runs.
func compareAudioLabels(a, b string) int {
	ak, bk := catalog.SortKey(a), catalog.SortKey(b)
	if catalog.NaturalKeyLess(ak, bk) {
		return -1
	}
	if catalog.NaturalKeyLess(bk, ak) {
		return 1
	}
	return 0
}
