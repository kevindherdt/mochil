package web

import (
	"net/http"
	"path"
	"strings"
)

// ServeHTTP is intentionally only a transport dispatcher. Feature behavior
// belongs to the focused HTTP adapters and application services it calls.
func (m *multiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !m.authorizeAdministrator(w, r) {
		return
	}
	if androidRoute(r.URL.Path) && !m.moduleOn("android") {
		http.NotFound(w, r)
		return
	}
	if module := moduleForPath(r.URL.Path); module != "" && !m.moduleOn(module) {
		m.moduleOffResponse(w, r)
		return
	}
	switch {
	case r.URL.Path == "/api/diagnostics":
		m.apiDiagnostics(w, r)
	case r.URL.Path == "/api/modules":
		m.apiModules(w, r)
	case isJellyfinPath(r.URL.Path):
		m.jellyfin(w, r)
	case r.URL.Path == "/next" || strings.HasPrefix(r.URL.Path, "/next/"):
		// The app used to live here. Old bookmarks, home-screen icons and links inside a stale APK still
		// ask for /next/…, so send them to the same path without the prefix rather than 404ing them.
		target := strings.TrimPrefix(r.URL.Path, "/next")
		if target == "" {
			target = "/"
		}
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	case r.URL.Path == "/editor/import":
		// The retired editor's own "Import" tab still points here, and so do bookmarks. Import is a real feature that
		// has not been ported yet, so the address has to keep working rather than landing on the app's Not-found.
		http.Redirect(w, r, "/legacy/editor/import", http.StatusPermanentRedirect)
	case r.URL.Path == "/hermes":
		// The legacy chat page is superseded by the app's AI screen.
		http.Redirect(w, r, "/lyana", http.StatusPermanentRedirect)
	case r.URL.Path == "/legacy" || strings.HasPrefix(r.URL.Path, "/legacy/"):
		// The retired shell, kept reachable while the last pages that still live in it (the spreadsheet
		// editor, managed import and the legacy chat) are either ported or dropped deliberately.
		m.legacyApp(w, r)
	case r.URL.Path == "/":
		m.nextApp(w, r)
	case r.URL.Path == "/sw.js" || r.URL.Path == "/manifest.webmanifest":
		m.pwaFile(w, r)
	case strings.HasPrefix(r.URL.Path, "/assets/"):
		m.assets(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/readium/"):
		m.readiumBook(w, r)
	case r.URL.Path == "/api/avatar" || strings.HasPrefix(r.URL.Path, "/api/avatar/"):
		m.apiAvatar(w, r)
	case r.URL.Path == "/api/hermes" || strings.HasPrefix(r.URL.Path, "/api/hermes/"):
		m.apiHermes(w, r)
	case strings.HasPrefix(r.URL.Path, "/reader/"):
		m.reader(w, r)
	case strings.HasPrefix(r.URL.Path, "/videos/"):
		m.streamVideo(w, r)
	case strings.HasPrefix(r.URL.Path, "/audio/"):
		m.streamAudio(w, r)
	case strings.HasPrefix(r.URL.Path, "/podcast-media/"):
		m.podcastMedia(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/podcast-artwork/"):
		m.podcastArtwork(w, r)
	case r.URL.Path == "/api/tmdb-portrait":
		m.tmdbPortrait(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-subtitles/"):
		m.videoSubtitle(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-hls/"):
		m.videoHLS(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-artwork/"):
		m.videoArtworkHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-metadata-artwork/"):
		m.videoMetadataArtworkHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-show-metadata-artwork/"):
		m.videoShowMetadataArtworkHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-metadata-search/"):
		m.apiVideoMetadataSearch(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-metadata-match/"):
		m.apiVideoMetadataMatch(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-metadata/"):
		m.apiVideoMetadata(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-show-metadata/"):
		m.apiVideoShowMetadata(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-details/"):
		m.apiVideoDetails(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/video-previews/"):
		m.videoPreviewHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/audio-artwork/"):
		m.audioArtworkHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/audio-hls/"):
		m.audioHLSHTTP(w, r)
	case r.URL.Path == "/api/libraries":
		m.apiLibraries(w, r)
	case r.URL.Path == "/api/health":
		m.apiHealth(w, r)
	case r.URL.Path == "/api/playback-policy":
		m.playbackPolicyHTTP(w, r)
	case r.URL.Path == "/api/jobs":
		m.apiJobs(w, r)
	case r.URL.Path == "/api/library-scan":
		m.apiLibraryScan(w, r)
	case r.URL.Path == "/api/videos":
		m.apiVideos(w, r)
	case r.URL.Path == "/api/audio":
		m.apiAudio(w, r)
	case r.URL.Path == "/api/audio-groups":
		m.androidAudioGroups(w, r)
	case r.URL.Path == "/api/audio-group-tracks":
		m.androidAudioGroupTracks(w, r)
	case r.URL.Path == "/api/audio-group-download":
		m.audioGroupDownload(w, r)
	case r.URL.Path == "/api/audio-artist-albums":
		m.androidArtistAlbums(w, r)
	case r.URL.Path == "/api/audiobook-series":
		m.apiAudiobookSeries(w, r)
	case r.URL.Path == "/api/audio-analysis":
		m.apiAudioAnalysis(w, r)
	case r.URL.Path == "/api/documents":
		m.apiDocuments(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/documents/"):
		m.apiDocumentText(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/vaults/"):
		m.apiVaultMutation(w, r)
	case r.URL.Path == "/api/wiki" || strings.HasPrefix(r.URL.Path, "/api/wiki/"):
		m.wikiAPI(w, r)
	case r.URL.Path == "/api/audio-queue":
		m.apiAudioQueue(w, r)
	case r.URL.Path == "/api/audio-playlists" || strings.HasPrefix(r.URL.Path, "/api/audio-playlists/"):
		m.apiAudioPlaylists(w, r)
	case r.URL.Path == "/api/audio-favorites":
		m.apiAudioFavorites(w, r)
	case r.URL.Path == "/api/podcast-feeds":
		m.apiPodcastFeeds(w, r)
	case r.URL.Path == "/api/podcast-opml":
		m.apiPodcastOPML(w, r)
	case r.URL.Path == "/api/podcast-search":
		m.apiPodcastSearch(w, r)
	case r.URL.Path == "/api/podcast-search-artwork":
		m.podcastSearchArtwork(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/podcast-feeds/"):
		m.apiPodcastFeed(w, r)
	case r.URL.Path == "/api/podcast-downloads" || strings.HasPrefix(r.URL.Path, "/api/podcast-downloads/"):
		m.apiPodcastDownload(w, r)
	case r.URL.Path == "/api/audio-progress" || strings.HasPrefix(r.URL.Path, "/api/audio-progress/"):
		m.apiAudioProgress(w, r)
	case r.URL.Path == "/api/audio-navigation":
		m.apiAudioNavigation(w, r)
	case r.URL.Path == "/api/video-shows":
		m.apiVideoShows(w, r)
	case r.URL.Path == "/api/video-progress" || strings.HasPrefix(r.URL.Path, "/api/video-progress/"):
		m.apiVideoProgress(w, r)
	case r.URL.Path == "/api/books":
		m.apiBooks(w, r)
	case r.URL.Path == "/api/book-navigation":
		m.apiBookNavigation(w, r)
	case r.URL.Path == "/api/series":
		m.apiSeries(w, r)
	case r.URL.Path == "/api/authors":
		m.apiAuthors(w, r)
	case r.URL.Path == "/api/metadata/issues":
		m.apiMetadataIssues(w, r)
	case r.URL.Path == "/api/metadata/music-quality":
		m.apiMusicQuality(w, r)
	case r.URL.Path == "/api/metadata/video-quality":
		m.apiVideoQuality(w, r)
	case r.URL.Path == "/api/metadata/ingest-preview":
		m.apiIngestPreview(w, r)
	case r.URL.Path == "/api/metadata/managed-ingest" || strings.HasPrefix(r.URL.Path, "/api/metadata/managed-ingest/"):
		m.apiManagedIngest(w, r)
	case r.URL.Path == "/api/metadata/suggestions":
		m.apiMetadataSuggestions(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/metadata/editor/"):
		// Import is a basic need, not the optional spreadsheet editor the rest of
		// this prefix belongs to; only the spreadsheet's own routes stay gated.
		route := strings.TrimPrefix(r.URL.Path, "/api/metadata/editor/")
		importRoute := route == "import" || route == "import-names" || route == "import-library"
		if !importRoute && !m.moduleOn("editor") {
			http.NotFound(w, r)
			return
		}
		m.apiEditor(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/metadata/hardcover/"):
		m.apiHardcover(w, r)
	case r.URL.Path == "/api/metadata/provider-connection":
		m.apiProviderConnection(w, r)
	case r.URL.Path == "/api/metadata/suggestion-artwork":
		m.metadataSuggestionArtworkHTTP(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/publication-metadata/"):
		m.apiPublicationMetadata(w, r)
	case r.URL.Path == "/api/metadata/jobs":
		m.apiMetadataJobs(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/metadata/jobs/"):
		m.apiMetadataJob(w, r)
	case r.URL.Path == "/api/changelog":
		m.apiChangelog(w, r)
	case r.URL.Path == "/api/collections" || strings.HasPrefix(r.URL.Path, "/api/collections/"):
		m.apiCollections(w, r)
	case r.URL.Path == "/api/external-library" || strings.HasPrefix(r.URL.Path, "/api/external-library/"):
		m.apiExternalLibrary(w, r)
	case r.URL.Path == "/api/reading":
		m.apiReading(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/reading/"):
		m.apiReadingBook(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/books/"):
		if strings.HasSuffix(r.URL.Path, "/sync") {
			m.apiBookSync(w, r)
		} else {
			m.apiBook(w, r)
		}
	case strings.HasPrefix(r.URL.Path, "/read/"):
		m.readBook(w, r)
	case strings.HasPrefix(r.URL.Path, "/covers/"):
		m.cover(w, r)
	case r.URL.Path == "/api/settings":
		m.settings(w, r)
	case r.URL.Path == "/api/folders":
		m.apiFolders(w, r)
	case r.URL.Path == "/api/library-inspect":
		m.apiLibraryInspect(w, r)
	case r.URL.Path == "/api/accounts" || strings.HasPrefix(r.URL.Path, "/api/accounts/"):
		m.accountAPI(w, r)
	case r.URL.Path == "/api/me":
		m.apiMe(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/narration/"):
		m.narrationAPI(w, r)
	case r.URL.Path == "/api/video-preferences":
		m.apiVideoPreferences(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/koreader/"):
		m.koReaderPlugin(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/annotations/"):
		m.annotationAPI(w, r)
	case r.URL.Path == "/api/android/v1/login":
		m.androidLoginAPI(w, r)
	case r.URL.Path == "/api/android/v1/pairing" || strings.HasPrefix(r.URL.Path, "/api/android/v1/pairing/"):
		m.androidPairingAPI(w, r)
	case r.URL.Path == "/pair":
		m.androidPairingPage(w, r)
	case r.URL.Path == "/api/android/app.apk":
		m.androidAppDownload(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/android/v1/"):
		m.androidAPI(w, r)
	case r.URL.Path == "/api/restart" && r.Method == http.MethodPost:
		m.restart(w)
	case r.URL.Path == "/healthz":
		m.root.health(w, r)
	case r.URL.Path == "/about":
		m.aboutText(w, r)
	case r.URL.Path == "/api/version":
		m.versionAPI(w, r)
	case m.kosync != nil && isKoSyncPath(r.URL.Path):
		m.serveKoSyncLogged(w, r)
	case r.URL.Path == "/opds":
		m.catalog(w, r)
	case r.URL.Path == "/opds/continue":
		m.opdsContinue(w, r)
	case m.collections != nil && strings.HasPrefix(r.URL.Path, "/opds/collections"):
		m.opdsCollections(w, r)
	case r.URL.Path == "/opds/":
		http.Redirect(w, r, m.root.href("/opds"), http.StatusPermanentRedirect)
	case strings.HasPrefix(r.URL.Path, "/opds/libraries/"):
		m.serveLibrary(w, r, "/opds/libraries/", true)
	case strings.HasPrefix(r.URL.Path, "/books/"):
		m.serveAcquisition(w, r)
	case strings.HasPrefix(r.URL.Path, "/documents/"):
		m.serveDocument(w, r)
	case strings.HasPrefix(r.URL.Path, "/stream/"):
		m.serveStream(w, r)
	case len(m.libraries) == 1 && strings.HasPrefix(r.URL.Path, "/opds/"):
		m.serveLegacy(w, r)
	default:
		// An unknown API path has to answer as an API. Handing it the app's HTML with a 200 makes a typo'd endpoint
		// look like a success to every client that parses JSON — the APK, KOReader, a script.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
			return
		}
		// A path that names a file (it has an extension) is a typo, not an app route — keep the branded 404
		// rather than answering a missing .js or .ico with an HTML shell. Everything else belongs to the app
		// router, which shows its own Not-found screen for a real dead end.
		if path.Ext(r.URL.Path) != "" {
			m.notFoundPage(w, r)
			return
		}
		m.nextApp(w, r)
	}
}
