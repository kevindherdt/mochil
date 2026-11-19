package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.catalogue.LibraryShellProjection
import net.kdhmedia.mochil.catalogue.shellProjection
import net.kdhmedia.mochil.dashboard.DashboardViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoViewModel
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map

/**
 * Owns Home-only state so dashboard refreshes and dismissals do not recompose the
 * connected app shell or unrelated destinations.
 */
@Composable
internal fun MobileDashboardRoute(
    dashboardViewModel: DashboardViewModel,
    libraryViewModel: LibraryViewModel,
    musicViewModel: MusicViewModel,
    podcastViewModel: PodcastViewModel,
    playbackViewModel: PlaybackViewModel,
    videoViewModel: VideoViewModel,
    showCollections: Boolean,
    showSources: Boolean,
    onOpenSettings: () -> Unit,
    onOpenLibrary: () -> Unit,
    onOpenCollections: () -> Unit,
    onOpenSources: () -> Unit,
    onWatch: (VideoItem) -> Unit,
) {
    val dashboardState by dashboardViewModel.state.collectAsStateWithLifecycle()
    val directoryFlow = remember(libraryViewModel) {
        libraryViewModel.state.map { it.shellProjection() }.distinctUntilChanged()
    }
    val directory by directoryFlow.collectAsStateWithLifecycle(LibraryShellProjection())

    Box {
    DashboardScreen(
        state = dashboardState,
        coverUrl = { libraryViewModel.coverUrl(it).orEmpty() },
        artworkUrl = { track ->
            if (track.libraryKind == "podcasts") podcastViewModel.artworkUrl(track)
            else musicViewModel.artworkUrl(track)
        },
        videoArtworkUrl = { item -> videoViewModel.artworkUrl(item.artworkPath) },
        authorizationHeader = libraryViewModel.authorizationHeader(),
        onRead = libraryViewModel::readBook,
        onDismissReading = dashboardViewModel::dismissReading,
        onResumeHeld = dashboardViewModel::resumeHeld,
        onListen = { track ->
            val saved = dashboardState.listening.firstOrNull {
                it.track.libraryId == track.libraryId && it.track.id == track.id
            }
            val playbackUrl = if (track.libraryKind == "podcasts") {
                podcastViewModel::playbackUrl
            } else {
                musicViewModel::playbackUrl
            }
            playbackViewModel.play(
                listOf(track),
                track,
                playbackUrl,
                ((saved?.positionSeconds ?: 0.0) * 1_000).toLong(),
            )
        },
        onDismissListening = dashboardViewModel::dismissListening,
        onWatch = onWatch,
        onDismissWatching = dashboardViewModel::dismissWatching,
        onRefresh = dashboardViewModel::refresh,
        onOpenSettings = onOpenSettings,
        libraries = directory.libraries,
        selectedLibrary = directory.selected,
        showCollections = showCollections,
        showSources = showSources,
        onOpenCollections = onOpenCollections,
        onOpenSources = onOpenSources,
        onSelectLibrary = { library ->
            libraryViewModel.select(library)
            onOpenLibrary()
        },
    )
    OpeningBannerHost(libraryViewModel, Modifier.align(Alignment.BottomCenter))
    }
}
