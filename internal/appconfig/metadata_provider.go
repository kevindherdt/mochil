package appconfig

import "strings"

// applyMetadataProviders settles libraries[].metadata_provider before
// validation. The field is no longer a user choice: an empty or "auto" value
// means "whatever this kind and these tokens allow", and "off"/"disabled" means
// no online metadata at all.
//
// A leftover explicit name is kept only while it still fits the library: an
// install that chose Hardcover for its books keeps Hardcover, but a name that
// no longer matches the kind or has lost its token falls back to the automatic
// choice instead of failing validation. Nothing in the settings page can edit
// this field any more, so a stale value must never be able to block a save.
func applyMetadataProviders(cfg *Config) {
	for i := range cfg.Libraries {
		library := &cfg.Libraries[i]
		explicit := strings.TrimSpace(library.MetadataProvider)
		switch strings.ToLower(explicit) {
		case "", "auto":
			library.MetadataProvider = cfg.automaticMetadataProvider(library.Kind)
		case "off", "disabled":
			library.MetadataProvider = ""
		default:
			if cfg.metadataProviderFits(library.Kind, explicit) {
				library.MetadataProvider = explicit
				continue
			}
			library.MetadataProvider = cfg.automaticMetadataProvider(library.Kind)
		}
	}
}

// metadataProviderFits reports whether a provider can answer for this kind and
// has the credential it needs.
func (cfg Config) metadataProviderFits(kind, provider string) bool {
	switch provider {
	case "hardcover":
		return isPublicationKind(kind) && strings.TrimSpace(cfg.MetadataProviders.HardcoverToken) != ""
	case "openlibrary":
		return isPublicationKind(kind)
	case "tmdb":
		return isVideoKind(kind) && strings.TrimSpace(cfg.MetadataProviders.TMDBToken) != ""
	}
	return false
}

func isVideoKind(kind string) bool {
	switch kind {
	case "movies", "tv", "anime":
		return true
	}
	return false
}

// automaticMetadataProvider is the provider a kind can actually reach: Open
// Library for publications, Hardcover for publications when a token is set,
// TMDB for video when a token is set, and nothing for everything else (Open
// Library cannot answer for video, and documents have no online catalogue).
func (cfg Config) automaticMetadataProvider(kind string) string {
	switch kind {
	case "books", "comics", "manga", "mixed":
		if strings.TrimSpace(cfg.MetadataProviders.HardcoverToken) != "" {
			return "hardcover"
		}
		return "openlibrary"
	case "movies", "tv", "anime":
		if strings.TrimSpace(cfg.MetadataProviders.TMDBToken) != "" {
			return "tmdb"
		}
	}
	return ""
}
