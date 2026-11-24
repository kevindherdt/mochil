#!/usr/bin/env bash
set -euo pipefail

declare -A grandfathered_lines=(
  [internal/web/static/reader.js]=360
  [android/app/src/main/java/net/kdhmedia/mochil/reader/epub/EpubReaderActivity.kt]=508
  [android/app/src/main/java/net/kdhmedia/mochil/ui/MusicScreen.kt]=320
  [android/app/src/main/java/net/kdhmedia/mochil/video/VideoPlayerActivity.kt]=475
)
declare -A grandfathered_bytes=(
  [internal/web/static/audio-library.js]=31600
)

failed=0
while IFS= read -r -d '' path; do
  [[ "$path" == */vendor/* || "$path" == */static_test/* || "$path" == *_test.go || "$path" == */static/next/* || "$path" == */static/dist/* ]] && continue
  lines=$(wc -l < "$path")
  bytes=$(wc -c < "$path")
  line_limit=${grandfathered_lines[$path]:-550}
  byte_limit=${grandfathered_bytes[$path]:-100000}
  if (( lines > line_limit )); then
    echo "$path grew to $lines lines (budget: $line_limit); extract a focused module" >&2
    failed=1
  fi
  if (( bytes > byte_limit )); then
    echo "$path grew to $bytes bytes (budget: $byte_limit); extract a focused module" >&2
    failed=1
  fi
done < <(find internal cmd android/app/src/main -type f \( -name '*.go' -o -name '*.js' -o -name '*.kt' -o -name '*.css' -o -name '*.html' \) -print0)

if [[ ! -f internal/web/metadata_provider_http.go ]] || grep -q 'apiMetadataSuggestions\|metadataSuggestionArtworkHTTP\|apiPublicationMetadata' internal/web/metadata_http.go; then
  echo "metadata review transactions must remain separate from provider lookup, artwork, and publication-cover HTTP" >&2
  failed=1
fi

if [[ ! -f internal/podcast/rss.go ]] || grep -Eq 'type rss struct|func parseRSS|func podcastPlainText' internal/podcast/store.go; then
  echo "podcast storage must delegate RSS parsing, sanitization, and date normalization to rss.go" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/ui/MusicGroupBrowser.kt ]] || grep -q 'fun GroupTracks' android/app/src/main/java/net/kdhmedia/mochil/ui/MusicScreen.kt; then
  echo "MusicScreen.kt must delegate artist and group detail rendering to MusicGroupBrowser.kt" >&2
  failed=1
fi
mobile_root=$(sed -n '1,/^[[:space:]]*AppShell(/p' android/app/src/main/java/net/kdhmedia/mochil/ui/MochilApp.kt)
if grep -q 'musicQueueViewModel.state.collectAsStateWithLifecycle\|musicPlaylistViewModel.state.collectAsStateWithLifecycle' <<<"$mobile_root" || [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/ui/MobilePlaybackHost.kt ]] || ! grep -q 'fun MobileExpandedPlayerRoute' android/app/src/main/java/net/kdhmedia/mochil/ui/MobilePlaybackHost.kt; then
  echo "mobile queue and playlist subscriptions must remain inside their consuming Library/player surfaces" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/catalogue/LibraryStateProjection.kt ]] || grep -q 'libraryViewModel.state.collectAsStateWithLifecycle' android/app/src/main/java/net/kdhmedia/mochil/ui/MobileDashboardRoute.kt android/app/src/main/java/net/kdhmedia/mochil/ui/MobileDeviceSettingsRoute.kt android/app/src/main/java/net/kdhmedia/mochil/ui/TvSettingsRoute.kt || ! grep -q 'visibilityProjection()' android/app/src/main/java/net/kdhmedia/mochil/ui/TvSettingsRoute.kt; then
  echo "Home and Settings must observe distinct library directory projections instead of catalogue state" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/ui/AuthenticatedArtwork.kt ]] || grep -q 'fun AuthenticatedArtwork' android/app/src/main/java/net/kdhmedia/mochil/ui/MusicScreen.kt; then
  echo "authorization-aware artwork loading must remain a shared UI primitive" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/video/VideoPlaybackRecovery.kt ]] || grep -q 'reconnectAttempts\|fallbackAttempted' android/app/src/main/java/net/kdhmedia/mochil/video/VideoPlayerActivity.kt; then
  echo "VideoPlayerActivity.kt must delegate retry and fallback ownership to VideoPlaybackRecovery.kt" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/ui/ReaderDeviceSettings.kt ]] || grep -q 'fun ReaderDefaultSettings\|fun ReaderHardwareSettings' android/app/src/main/java/net/kdhmedia/mochil/ui/DeviceSettingsScreen.kt; then
  echo "DeviceSettingsScreen.kt must delegate reader defaults and hardware controls to ReaderDeviceSettings.kt" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/catalogue/LibraryNavigationStore.kt ]] || grep -q 'getSharedPreferences("library_navigation"' android/app/src/main/java/net/kdhmedia/mochil/catalogue/LibraryViewModel.kt; then
  echo "LibraryViewModel.kt must delegate durable navigation and cache persistence to LibraryNavigationStore.kt" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/catalogue/PublicationCatalogueRepository.kt ]] || ! grep -q 'PublicationCatalogueRepository' android/app/src/main/java/net/kdhmedia/mochil/catalogue/LibraryViewModel.kt || grep -Eq 'CatalogueClient\(|CollectionClient\(|withContext\(Dispatchers\.IO\)' android/app/src/main/java/net/kdhmedia/mochil/catalogue/LibraryViewModel.kt; then
  echo "LibraryViewModel.kt must delegate publication and collection transport to PublicationCatalogueRepository.kt" >&2
  failed=1
fi
if ! grep -q 'LaunchedEffect(contentRouteKey' android/app/src/main/java/net/kdhmedia/mochil/ui/TvLibraryScreen.kt || ! grep -q 'tvContentRouteKey(library.selected?.id' android/app/src/main/java/net/kdhmedia/mochil/ui/TvLibraryScreen.kt; then
  echo "TV library focus recovery must include the selected shelf in its content route identity" >&2
  failed=1
fi
if ! grep -q 'explicitSeekForReplay(System.currentTimeMillis())' android/app/src/main/java/net/kdhmedia/mochil/playback/PlaybackService.kt || ! grep -q 'EXPLICIT_SEEK_REPLAY_WINDOW_MS' android/app/src/main/java/net/kdhmedia/mochil/playback/AudioProgressJournal.kt; then
  echo "durable audio seek replay must use bounded explicit ownership instead of a permanent stale-write override" >&2
  failed=1
fi
if ! grep -q 'decideEpubInitialPosition' android/app/src/main/java/net/kdhmedia/mochil/reader/epub/EpubProgressCoordinator.kt || ! grep -q '!localPending.*UseRemote' android/app/src/main/java/net/kdhmedia/mochil/reader/epub/EpubProgressCoordinator.kt || ! grep -q 'sameDevice.*KeepLocal' android/app/src/main/java/net/kdhmedia/mochil/reader/epub/EpubProgressCoordinator.kt; then
  echo "EPUB reconciliation must preserve pending offline progress from the same device" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/reader/FixedLayoutLocalProgressStore.kt ]] || ! grep -q 'localProgress.saveNow(progressKey' android/app/src/main/java/net/kdhmedia/mochil/reader/pdf/PdfReaderActivity.kt || ! grep -q 'localProgress.saveNow(progressKey' android/app/src/main/java/net/kdhmedia/mochil/reader/comic/ComicReaderActivity.kt; then
  echo "PDF and comic readers must synchronously commit their latest local page on pause" >&2
  failed=1
fi
if ! grep -q 'fixedLayoutSyncKey(libraryId, bookId)' android/app/src/main/java/net/kdhmedia/mochil/reader/pdf/PdfProgressViewModel.kt || ! grep -q 'store.migrateLegacyKey(legacyKey, key)' android/app/src/main/java/net/kdhmedia/mochil/reader/pdf/PdfProgressViewModel.kt; then
  echo "PDF and comic synchronization must use collision-safe local identities and migrate legacy progress" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/reader/epub/EpubReadingProgressChrome.kt ]] || grep -q 'fun EpubPersistentProgress\|fun ReadingStatisticsPanel' android/app/src/main/java/net/kdhmedia/mochil/reader/epub/EpubReaderChrome.kt; then
  echo "EpubReaderChrome.kt must delegate persistent progress and learned reading statistics presentation" >&2
  failed=1
fi
if ! grep -q 'showsShuffle(track.libraryKind)' android/app/src/main/java/net/kdhmedia/mochil/ui/TvNowPlayingScreen.kt || ! grep -q 'showsPlaybackSpeed(track.libraryKind)' android/app/src/main/java/net/kdhmedia/mochil/ui/TvNowPlayingScreen.kt; then
  echo "TV Now Playing must reserve shuffle for music and playback speed for spoken media" >&2
  failed=1
fi
if ! grep -q 'Text("Book details"' android/app/src/main/java/net/kdhmedia/mochil/ui/CollectionPickerSheet.kt || grep -q 'onShowDetails' android/app/src/main/java/net/kdhmedia/mochil/ui/PublicationScreen.kt; then
  echo "publication cards must expose details through the long-press options sheet, not duplicated card chrome" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/ui/PublicationItems.kt ]] || grep -q 'fun BookGrid\|fun BookDetail\|fun PublicationArtwork' android/app/src/main/java/net/kdhmedia/mochil/ui/PublicationScreen.kt; then
  echo "PublicationScreen.kt must delegate reusable item, detail, artwork, and paging presentation to PublicationItems.kt" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/podcast/PodcastDownloadCoordinator.kt ]] || ! grep -q 'PodcastDownloadCoordinator' android/app/src/main/java/net/kdhmedia/mochil/podcast/PodcastViewModel.kt || grep -Eq 'OfflineAudioStore|PodcastDevicePolicyStore|ConnectivityManager|reconcilePodcastEpisodes|recoverExternalDownloads|podcastDeviceDownloadPlan' android/app/src/main/java/net/kdhmedia/mochil/podcast/PodcastViewModel.kt; then
  echo "PodcastViewModel.kt must delegate device download recovery, matching, retention, and network policy to PodcastDownloadCoordinator.kt" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/audio/MusicCatalogueRepository.kt ]] || ! grep -q 'MusicCatalogueRepository' android/app/src/main/java/net/kdhmedia/mochil/audio/MusicViewModel.kt || grep -Eq 'PerformanceDiagnostics|client\.(groups|tracks|groupTracks|artistAlbums|favorites|recentlyPlayed|navigation|setFavorite)' android/app/src/main/java/net/kdhmedia/mochil/audio/MusicViewModel.kt; then
  echo "MusicViewModel.kt must delegate measured remote catalogue queries and favorite writes to MusicCatalogueRepository.kt" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/offline/ResumableHttpDownload.kt ]] || ! grep -q 'ResumableHttpDownload.copyResponse' android/app/src/main/java/net/kdhmedia/mochil/catalogue/PublicationOpener.kt || ! grep -q 'ResumableHttpDownload.copyResponse' android/app/src/main/java/net/kdhmedia/mochil/audio/MusicDownloadStore.kt; then
  echo "publication and audio downloads must share validator-safe HTTP range recovery" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/wiki/WikiRepository.kt ]] || ! grep -q 'WikiRepository(client)' android/app/src/main/java/net/kdhmedia/mochil/wiki/WikiViewModel.kt || grep -Eq 'client\.(tree|search|page|save|createFolder|createPage|movePage|trashPage|backlinks)' android/app/src/main/java/net/kdhmedia/mochil/wiki/WikiViewModel.kt; then
  echo "WikiViewModel.kt must delegate compound remote vault operations to WikiRepository.kt" >&2
  failed=1
fi
if ! grep -q 'fun audioTrackIdentityKey' android/app/src/main/java/net/kdhmedia/mochil/audio/AudioModels.kt || grep -REq '"\$libraryId:\$id"|"\$\{track\.libraryId\}:\$\{track\.id\}"' android/app/src/main/java/net/kdhmedia/mochil/{audio,podcast,playback}; then
  echo "audio state, downloads, queues, and playback must share the collision-safe AudioTrack identity" >&2
  failed=1
fi
if ! grep -q 'fixedLayoutStorageKey' android/app/src/main/java/net/kdhmedia/mochil/reader/pdf/PdfReaderActivity.kt || ! grep -q 'fixedLayoutStorageKey' android/app/src/main/java/net/kdhmedia/mochil/reader/comic/ComicReaderActivity.kt || ! grep -q 'localProgress.migrateLegacyKey' android/app/src/main/java/net/kdhmedia/mochil/reader/pdf/PdfReaderActivity.kt || ! grep -q 'localProgress.migrateLegacyKey' android/app/src/main/java/net/kdhmedia/mochil/reader/comic/ComicReaderActivity.kt; then
  echo "PDF and comic exact local page state must use collision-safe identities and migrate legacy keys" >&2
  failed=1
fi
if ! grep -q 'fun videoPlaybackIdentityKey' android/app/src/main/java/net/kdhmedia/mochil/video/VideoPlaybackQueue.kt || ! grep -q 'setMediaId(videoPlaybackIdentityKey' android/app/src/main/java/net/kdhmedia/mochil/video/VideoMediaItems.kt || grep -q 'recovery.transition("\$libraryId:\$itemId")' android/app/src/main/java/net/kdhmedia/mochil/video/VideoPlayerActivity.kt; then
  echo "video Media3 items and retry ownership must share a collision-safe playback identity" >&2
  failed=1
fi
if [[ ! -f android/app/src/main/java/net/kdhmedia/mochil/catalogue/CatalogueIdentity.kt ]] || ! grep -q 'fun catalogueItemIdentityKey' android/app/src/main/java/net/kdhmedia/mochil/catalogue/CatalogueIdentity.kt || grep -REq '"\$\{it\.libraryId\}:\$\{it\.(id|bookId|name)\}"' android/app/src/main/java/net/kdhmedia/mochil/ui; then
  echo "Compose catalogue rows must use the shared collision-safe library/item identity" >&2
  failed=1
fi
if [[ ! -f internal/web/static/reading-position-equivalence.js ]] || ! grep -q "from './reading-position-equivalence.js" internal/web/static/reader-position-sync.js || ! grep -q 'sameReflowableLocation' internal/reading/model.go || ! grep -q 'equivalentProgress(current, value)' internal/externallibrary/store.go; then
  echo "reading progress must share semantic same-location acknowledgements across server and web clients" >&2
  failed=1
fi

exit "$failed"
