package net.kdhmedia.mochil.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.dashboard.DashboardViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.video.VideoItem
import net.kdhmedia.mochil.video.VideoViewModel

/** Owns TV Home state, resume actions, and entry-focus restoration. */
@Composable
internal fun TvHomeRoute(
    libraryViewModel: LibraryViewModel,
    musicViewModel: MusicViewModel,
    playbackViewModel: PlaybackViewModel,
    podcastViewModel: PodcastViewModel,
    videoViewModel: VideoViewModel,
    dashboardViewModel: DashboardViewModel,
    focusRail: () -> Unit,
    onPlayVideo: (VideoItem) -> Unit,
) {
    val dashboard by dashboardViewModel.state.collectAsStateWithLifecycle()
    TvHomeScreen(
        state = dashboard,
        artworkUrl = {
            if (it.libraryKind == "podcasts") podcastViewModel.artworkUrl(it)
            else musicViewModel.artworkUrl(it)
        },
        videoArtworkUrl = { videoViewModel.artworkUrl(it.artworkPath) },
        videoBackdropUrl = { item -> item.backdropPath.takeIf(String::isNotBlank)?.let(videoViewModel::artworkUrl).orEmpty() },
        authorizationHeader = libraryViewModel.authorizationHeader(),
        onListen = { item ->
            dashboardViewModel.resolveListening(item) { track ->
                playbackViewModel.play(
                    listOf(track),
                    track,
                    if (track.libraryKind == "podcasts") podcastViewModel::playbackUrl else musicViewModel::playbackUrl,
                    (track.resumePositionSeconds * 1_000).toLong(),
                )
            }
        },
        onWatch = onPlayVideo,
        onDismissListening = dashboardViewModel::dismissListening,
        onDismissWatching = dashboardViewModel::dismissWatching,
        onRetry = dashboardViewModel::refresh,
        focusRail = focusRail,
    )
}
