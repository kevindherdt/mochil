package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.Row
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.foundation.layout.Box
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.StateFlow
import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.AuthorSummary
import net.kdhmedia.mochil.catalogue.DownloadQueueItem
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.LibrarySummary
import net.kdhmedia.mochil.catalogue.PublicationCollection
import net.kdhmedia.mochil.catalogue.PublicationGrouping
import net.kdhmedia.mochil.catalogue.PublicationLayout
import net.kdhmedia.mochil.catalogue.SeriesSummary
import net.kdhmedia.mochil.catalogue.isPublicationLibrary
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoShow
import net.kdhmedia.mochil.video.VideoState

data class VideoBindings(
    val state: VideoState,
    val open: (String, String) -> Unit,
    val artworkUrl: (String) -> String,
    val authorizationHeader: String?,
    val openShow: (VideoShow) -> Unit,
    val closeShow: () -> Unit,
    val selectSeason: (Int) -> Unit,
    val setLayout: (net.kdhmedia.mochil.video.VideoLayout) -> Unit,
    val setSearchOpen: (Boolean) -> Unit,
    val search: (String) -> Unit,
    val play: (VideoItem) -> Unit,
    val dismissContinueWatching: (VideoItem) -> Unit,
    val setWatched: (VideoItem, Boolean) -> Unit,
    val loadMoreShows: () -> Unit,
    val loadMoreItems: () -> Unit,
    val retry: () -> Unit,
    val loadMovieDetails: (VideoItem) -> Unit = {},
)

@Composable
fun LibraryScreen(
    state: LibraryState,
    musicState: MusicState,
    podcasts: PodcastBindings,
    videos: VideoBindings,
    wiki: WikiBindings,
    coverUrl: (BookSummary) -> String?,
    seriesCoverUrl: (SeriesSummary) -> String?,
    authorCoverUrl: (AuthorSummary) -> String?,
    collectionCoverUrl: (PublicationCollection) -> String?,
    authorizationHeader: String?,
    onSelectLibrary: (LibrarySummary) -> Unit,
    onPublicationGrouping: (PublicationGrouping) -> Unit,
    onPublicationSort: (String) -> Unit,
    onPublicationLayout: (PublicationLayout) -> Unit,
    onOpenSeries: (SeriesSummary) -> Unit,
    onCloseSeries: () -> Unit,
    onOpenAuthor: (AuthorSummary) -> Unit,
    onCloseAuthor: () -> Unit,
    onOpenCollection: (net.kdhmedia.mochil.catalogue.PublicationCollection) -> Unit,
    onCloseCollection: () -> Unit,
    onRetrySeries: () -> Unit,
    onReadBook: (BookSummary) -> Unit,
    onQueueBook: (BookSummary) -> Unit,
    downloadQueue: StateFlow<List<DownloadQueueItem>>,
    onCloseBook: () -> Unit,
    onSearchPublications: (String) -> Unit,
    onPublicationBucket: (String) -> Unit,
    onDownloadedOnly: (Boolean) -> Unit,
    publicationCollections: PublicationCollectionActions,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
    onOpenMusic: (String, String) -> Unit,
    audioArtworkUrl: (AudioTrack) -> String,
    audioGroupArtworkUrl: (AudioGroup) -> String,
    onOpenAudioGroup: (AudioGroup) -> Unit,
    onOpenAudioAlbum: (AudioGroup) -> Unit,
    onCloseAudioGroup: () -> Unit,
    onCloseAudioArtist: () -> Unit,
    onMusicGrouping: (MusicGrouping) -> Unit,
    onMusicLayout: (MusicLayout) -> Unit,
    onSearchMusic: (String) -> Unit,
    onDownloadedMusicOnly: (Boolean) -> Unit,
    onFavoriteMusicOnly: (Boolean) -> Unit,
    onShowAllMusic: () -> Unit,
    onToggleMusicDownload: (AudioTrack) -> Unit,
    onToggleMusicFavorite: (AudioTrack) -> Unit,
    onPlayMusicNext: (AudioTrack) -> Unit,
    onAddMusicToQueue: (AudioTrack) -> Unit,
    musicPlaylists: MusicPlaylistBindings,
    onDownloadMusicGroup: () -> Unit,
    onQueueMusicGroup: (List<AudioTrack>, Boolean) -> Unit,
    onPlayMusicGroup: (List<AudioTrack>, Boolean) -> Unit,
    onPlayMusicArtist: (Boolean) -> Unit,
    onLoadMoreAudio: () -> Unit,
    onRetryAudio: () -> Unit,
    onPlayAudio: (AudioTrack) -> Unit,
    onOpenSettings: () -> Unit,
    onOpenHome: () -> Unit,
    showCollections: Boolean,
    showSources: Boolean,
    onOpenCollections: () -> Unit,
    onOpenSources: () -> Unit,
) {
    var headerActions by remember { mutableStateOf<List<HeaderAction>?>(null) }
    val expandedNavigation = LocalExpandedMobileNavigation.current
    val headerSlot = remember(expandedNavigation) { if (expandedNavigation) null else { content: List<HeaderAction>? -> headerActions = content } }
    androidx.compose.runtime.CompositionLocalProvider(LocalHeaderActions provides headerSlot) {
    Column(Modifier.fillMaxSize()) {
            LibraryHeader(state, onSelectLibrary, onRetry, onOpenSettings, onOpenHome, showCollections, showSources, onOpenCollections, onOpenSources, headerActions)
            when {
                state.initialLoading -> AccessibleLoadingIndicator("Loading libraries", Modifier.fillMaxSize())
                state.error != null && state.books.isEmpty() && state.series.isEmpty() && state.authors.isEmpty() -> CenteredStatus {
                    Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Button(onClick = onRetry) { Text("Try again") }
                }
                state.selected?.kind?.lowercase() in setOf("music", "audiobooks") -> {
                    val selected = requireNotNull(state.selected)
                    LaunchedEffect(selected.id) { onOpenMusic(selected.id, selected.kind.lowercase()) }
                    MusicScreen(
                        musicState, audioArtworkUrl, audioGroupArtworkUrl, authorizationHeader,
                        onPlayAudio, onOpenAudioGroup, onOpenAudioAlbum, onCloseAudioGroup, onCloseAudioArtist, onMusicGrouping,
                        onMusicLayout, onLoadMoreAudio, onRetryAudio,
                        onSearchMusic, onDownloadedMusicOnly, onFavoriteMusicOnly, onShowAllMusic, onToggleMusicDownload, onToggleMusicFavorite, onPlayMusicNext, onAddMusicToQueue, onDownloadMusicGroup, musicPlaylists, onQueueMusicGroup, onPlayMusicGroup, onPlayMusicArtist,
                    )
                }
                state.selected?.kind?.lowercase() == "podcasts" -> {
                    val selected = requireNotNull(state.selected)
                    LaunchedEffect(selected.id) { podcasts.open(selected.id) }
                    PodcastScreen(
                        podcasts.state, podcasts.showArtworkUrl, podcasts.episodeArtworkUrl, authorizationHeader,
                        podcasts.openShow, podcasts.closeShow, podcasts.setLayout, podcasts.search, podcasts.setPlayedFilter, podcasts.setEpisodeSort, podcasts.setPlayed,
                        podcasts.play, podcasts.toggleDownload, podcasts.loadMore, podcasts.retry, podcasts.subscribe, podcasts.unsubscribe,
                        podcasts.discover, podcasts.clearDiscovery, podcasts.discoveryArtworkUrl, podcasts.refreshShow, podcasts.updatePolicy,
                        podcasts.devicePolicy, podcasts.updateDevicePolicy,
                    )
                }
                state.selected?.kind?.lowercase() in setOf("movies", "tv", "anime", "video", "videos") -> {
                    val selected = requireNotNull(state.selected)
                    LaunchedEffect(selected.id) { videos.open(selected.id, selected.kind.lowercase()) }
                    VideoScreen(
                        videos.state, videos.artworkUrl, videos.authorizationHeader, videos.openShow, videos.closeShow,
                        videos.selectSeason, videos.setLayout, videos.setSearchOpen, videos.search, videos.play, videos.dismissContinueWatching, videos.setWatched, videos.loadMoreShows, videos.loadMoreItems, videos.retry, videos.loadMovieDetails,
                    )
                }
                state.selected?.kind?.lowercase() == "vault" -> {
                    val selected = requireNotNull(state.selected)
                    LaunchedEffect(selected.id) { wiki.open(selected.id) }
                    WikiScreen(wiki)
                }
                state.selected?.isPublicationLibrary() == true -> PublicationScreen(
                    state = state,
                    coverUrl = coverUrl,
                    seriesCoverUrl = seriesCoverUrl,
                    authorCoverUrl = authorCoverUrl,
                    collectionCoverUrl = collectionCoverUrl,
                    authorizationHeader = authorizationHeader,
                    onGrouping = onPublicationGrouping,
                    onSort = onPublicationSort,
                    onLayout = onPublicationLayout,
                    onOpenSeries = onOpenSeries,
                    onCloseSeries = onCloseSeries,
                    onOpenAuthor = onOpenAuthor,
                    onCloseAuthor = onCloseAuthor,
                    onOpenCollection = onOpenCollection,
                    onCloseCollection = onCloseCollection,
                    onRetrySeries = onRetrySeries,
                    onReadBook = onReadBook,
                    onQueueBook = onQueueBook,
                    downloadQueue = downloadQueue,
                    onCloseBook = onCloseBook,
                    onSearch = onSearchPublications,
                    onBucket = onPublicationBucket,
                    onDownloadedOnly = onDownloadedOnly,
                    collectionActions = publicationCollections,
                    onLoadMore = onLoadMore,
                    onRetry = onRetry,
                )
                state.selected != null -> CenteredStatus {
                    Text("${state.selected.name} is connected.", style = MaterialTheme.typography.titleMedium)
                    Text("Native ${state.selected.kind} browsing is coming next.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                else -> CenteredStatus { Text("No libraries are available.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            }
    }
    }
}

@Composable
private fun LibraryHeader(state: LibraryState, onSelect: (LibrarySummary) -> Unit, onRetry: () -> Unit, onOpenSettings: () -> Unit, onOpenHome: () -> Unit, showCollections: Boolean, showSources: Boolean, onOpenCollections: () -> Unit, onOpenSources: () -> Unit, actions: List<HeaderAction>?) {
    Column(Modifier.fillMaxWidth().padding(horizontal = MOBILE_CONTENT_GUTTER)) {
        if (!LocalExpandedMobileNavigation.current) Row(verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.weight(1f)) { MobileLibraryRail(state.libraries, state.selected, MobileShellDestination.Library, onOpenHome, showCollections, showSources, onOpenCollections, onOpenSources, onOpenSettings, onSelect) }
            actions?.let { HeaderActionButtons(it) }
        }
        if (state.offlineOnly) Row(verticalAlignment = Alignment.CenterVertically) {
            Text(listOfNotNull("Offline · local downloads only", state.offlineDiagnostic).joinToString("\n"), Modifier.weight(1f), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
            TextButton(onClick = onRetry) { Text("Retry") }
        }
    }
}

@Composable
private fun CenteredStatus(content: @Composable ColumnScope.() -> Unit) {
    Surface(Modifier.fillMaxSize()) {
        Column(
            Modifier.fillMaxSize().padding(32.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically),
            horizontalAlignment = Alignment.CenterHorizontally,
            content = content,
        )
    }
}
