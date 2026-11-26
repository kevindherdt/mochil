package net.kdhmedia.mochil.ui

import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicPlaylistState
import net.kdhmedia.mochil.audio.MusicPlaylistViewModel
import net.kdhmedia.mochil.audio.MusicQueueState
import net.kdhmedia.mochil.audio.MusicQueueViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel

/** Playback clocks recompose these small hosts, never the catalogue shell. */
@Composable
internal fun MobileMiniPlayerHost(playbackViewModel: PlaybackViewModel, navigationInset: Boolean = true, onExpand: () -> Unit) {
    val dock by playbackViewModel.dockState.collectAsStateWithLifecycle()
    MiniPlayer(dock, playbackViewModel.timelineState, playbackViewModel::toggle, playbackViewModel::closePlayback, onExpand, navigationInset)
}

/** Collects queue and playlist mutations only while their expanded surface exists. */
@Composable
internal fun MobileExpandedPlayerRoute(
    expanded: Boolean,
    panel: String,
    playbackViewModel: PlaybackViewModel,
    queueViewModel: MusicQueueViewModel,
    playlistViewModel: MusicPlaylistViewModel,
    playbackUrl: suspend (AudioTrack) -> String,
    artworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    onPanel: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    if (!expanded) return
    val queue by queueViewModel.state.collectAsStateWithLifecycle()
    val playlists by playlistViewModel.state.collectAsStateWithLifecycle()
    MobileExpandedPlayerHost(
        expanded = true,
        panel = panel,
        playbackViewModel = playbackViewModel,
        artworkUrl = artworkUrl,
        authorizationHeader = authorizationHeader,
        queueState = queue,
        onPlayQueued = { track -> playbackViewModel.play(queue.tracks, track, playbackUrl) },
        onMoveQueued = queueViewModel::move,
        onRemoveQueued = queueViewModel::remove,
        onClearQueue = queueViewModel::clear,
        playlistState = playlists,
        onPlayPlaylist = { name ->
            val tracks = playlistViewModel.playbackTracks(name)
            check(tracks.isNotEmpty()) { "This playlist has no available tracks." }
            if (tracks.isNotEmpty()) {
                queueViewModel.replace(tracks)
                playbackViewModel.play(tracks, tracks.first(), playbackUrl)
            }
        },
        onSaveQueueAsPlaylist = playlistViewModel::saveQueue,
        onDeletePlaylist = playlistViewModel::delete,
        onQueueNeeded = queueViewModel::ensureLoaded,
        onRetryQueue = queueViewModel::refresh,
        onPanel = onPanel,
        onDismiss = onDismiss,
    )
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
internal fun MobileExpandedPlayerHost(
    expanded: Boolean,
    panel: String,
    playbackViewModel: PlaybackViewModel,
    artworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    queueState: MusicQueueState,
    onPlayQueued: (AudioTrack) -> Unit,
    onMoveQueued: (Int, Int) -> Unit,
    onRemoveQueued: (Int) -> Unit,
    onClearQueue: () -> Unit,
    playlistState: MusicPlaylistState,
    onPlayPlaylist: suspend (String) -> Unit,
    onSaveQueueAsPlaylist: (String, List<AudioTrack>) -> Unit,
    onDeletePlaylist: (String) -> Unit,
    onQueueNeeded: () -> Unit,
    onRetryQueue: () -> Unit,
    onPanel: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    if (!expanded) return
    LaunchedEffect(panel) { if (panel == "queue" || panel == "playlists") onQueueNeeded() }
    val track by playbackViewModel.currentTrack.collectAsStateWithLifecycle()
    val currentTrack = track
    if (currentTrack == null) {
        LaunchedEffect(Unit) { onDismiss() }
        return
    }
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        when (panel) {
            "queue" -> MusicQueuePanel(queueState, onPlayQueued, onMoveQueued, onRemoveQueued, onClearQueue, { onPanel("playlists") }, onRetryQueue) { onPanel("player") }
            "playlists" -> MusicPlaylistsPanel(
                playlistState, queueState.tracks,
                onPlay = { onPlayPlaylist(it); onPanel("player") },
                onSaveQueue = onSaveQueueAsPlaylist,
                onDelete = onDeletePlaylist,
                onBack = { onPanel("queue") },
            )
            else -> MobileFullPlayer(
                playbackViewModel,
                artworkUrl(currentTrack),
                authorizationHeader,
                { onPanel("queue") },
            )
        }
    }
}

@Composable
private fun MobileFullPlayer(
    playbackViewModel: PlaybackViewModel,
    artworkUrl: String,
    authorizationHeader: String?,
    onOpenQueue: () -> Unit,
) {
    val state by playbackViewModel.playerState.collectAsStateWithLifecycle()
    FullPlayer(
        state,
        playbackViewModel.timelineState,
        artworkUrl,
        authorizationHeader,
        playbackViewModel::toggle,
        playbackViewModel::previous,
        playbackViewModel::next,
        playbackViewModel::seekTo,
        onOpenQueue,
        playbackViewModel::toggleRepeat,
        playbackViewModel::toggleShuffle,
        playbackViewModel::cycleSpeed,
        playbackViewModel::setSleepTimer,
        playbackViewModel::addAudiobookBookmark,
        playbackViewModel::removeAudiobookBookmark,
    )
}
