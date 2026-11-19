package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Repeat
import androidx.compose.material.icons.filled.RepeatOne
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Shuffle
import androidx.compose.material.icons.filled.SkipNext
import androidx.compose.material.icons.filled.SkipPrevious
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.media3.common.Player
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.playback.PlaybackViewModel

/** Remote-first full transport kept separate from catalogue browsing. */
@Composable
internal fun TvNowPlayingScreen(
    playbackViewModel: PlaybackViewModel,
    artworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    onPersistQueue: (List<AudioTrack>) -> Unit,
    onBack: () -> Unit,
) {
    val transport by playbackViewModel.transportState.collectAsStateWithLifecycle()
    val track = transport.current
    var showingQueue by remember { mutableStateOf(false) }
    val navigateBack = {
        navigateTvNowPlayingBack(showingQueue && track != null, { showingQueue = false }, onBack)
    }
    BackHandler(onBack = navigateBack)
    if (track == null) {
        val emptyBackFocus = remember { FocusRequester() }
        LaunchedEffect(Unit) { emptyBackFocus.requestFocus() }
        Column(
            Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(18.dp, Alignment.CenterVertically),
        ) {
            Text(
                "Nothing is playing.",
                style = MaterialTheme.typography.titleLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            TvIconButton(
                Icons.AutoMirrored.Filled.ArrowBack,
                "Back to browsing",
                onBack,
                Modifier.focusRequester(emptyBackFocus),
            )
        }
        return
    }
    TvBackdropEffect(artworkUrl(track))
    val transportFocus = remember { FocusRequester() }
    val focusSession = remember { TvNowPlayingFocusSession() }
    LaunchedEffect(track.libraryId, track.id, showingQueue) {
        if (focusSession.shouldFocusTransport(showingQueue)) transportFocus.requestFocus()
    }
    Column(Modifier.fillMaxSize().padding(horizontal = 56.dp, vertical = 24.dp), verticalArrangement = Arrangement.spacedBy(18.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TvIconButton(Icons.AutoMirrored.Filled.ArrowBack, "Back", navigateBack)
            Text("Now playing", Modifier.padding(start = 10.dp), style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Medium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.weight(1f))
            TvTextButton("Queue", { showingQueue = !showingQueue }, active = showingQueue)
            TvIconButton(Icons.Default.Close, "Close player", onClick = {
                closeTvNowPlaying(playbackViewModel::closePlayback, onBack)
            })
        }
        if (showingQueue) {
            TvPlaybackQueue(
                playbackViewModel = playbackViewModel,
                artworkUrl = artworkUrl,
                authorizationHeader = authorizationHeader,
                onPersistQueue = onPersistQueue,
            )
            return@Column
        }
        Row(Modifier.weight(1f), horizontalArrangement = Arrangement.spacedBy(42.dp), verticalAlignment = Alignment.CenterVertically) {
            AuthenticatedArtwork(artworkUrl(track), authorizationHeader, track.title, Modifier.size(310.dp).aspectRatio(1f), cornerRadius = 16.dp)
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                Column(verticalArrangement = Arrangement.spacedBy(5.dp)) {
                    Text(track.title, style = MaterialTheme.typography.headlineLarge, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    Text(track.artist.ifBlank { if (track.libraryKind == "podcasts") "Podcast" else "Unknown artist" }, style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (track.album.isNotBlank()) Text(track.album, style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
                TvNowPlayingTimeline(playbackViewModel, track.identityKey)
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                    if (showsShuffle(track.libraryKind)) {
                        TvIconButton(Icons.Default.Shuffle, if (transport.shuffleEnabled) "Disable shuffle" else "Enable shuffle", playbackViewModel::toggleShuffle, active = transport.shuffleEnabled, selectedState = transport.shuffleEnabled)
                    } else if (showsPlaybackSpeed(track.libraryKind)) {
                        val speedDescription = stringResource(net.kdhmedia.mochil.R.string.tv_playback_speed_description, playbackSpeedLabel(transport.speed))
                        TvTextButton("${playbackSpeedLabel(transport.speed)}×", playbackViewModel::cycleSpeed, Modifier.semantics { contentDescription = speedDescription })
                    }
                    TvIconButton(Icons.Default.SkipPrevious, "Previous", playbackViewModel::previous)
                    TvIconButton(if (transport.isPlaying) Icons.Default.Pause else Icons.Default.PlayArrow, if (transport.isPlaying) "Pause" else "Play", playbackViewModel::toggle, Modifier.focusRequester(transportFocus))
                    TvIconButton(Icons.Default.SkipNext, "Next", playbackViewModel::next)
                    TvIconButton(
                        if (transport.repeatMode == Player.REPEAT_MODE_ONE) Icons.Default.RepeatOne else Icons.Default.Repeat,
                        when (transport.repeatMode) { Player.REPEAT_MODE_ALL -> "Repeat all enabled"; Player.REPEAT_MODE_ONE -> "Repeat one enabled"; else -> "Enable repeat" },
                        playbackViewModel::toggleRepeat,
                        active = transport.repeatMode != Player.REPEAT_MODE_OFF,
                        selectedState = transport.repeatMode != Player.REPEAT_MODE_OFF,
                    )
                }
            }
        }
    }
}

@Composable
private fun TvNowPlayingTimeline(playbackViewModel: PlaybackViewModel, itemKey: String) {
    val timeline by playbackViewModel.timelineState.collectAsStateWithLifecycle()
    var dragged by remember(itemKey) { mutableFloatStateOf(-1f) }
    val duration = timeline.durationMs.coerceAtLeast(1L)
    val displayed = if (dragged >= 0f) dragged.toLong() else timeline.positionMs.coerceIn(0L, duration)
    val seekLabel = stringResource(net.kdhmedia.mochil.R.string.tv_playback_position)
    val seekState = if (timeline.durationMs > 0) {
        stringResource(net.kdhmedia.mochil.R.string.tv_playback_position_known,
            formatNowPlayingTime(displayed), formatNowPlayingTime(timeline.durationMs))
    } else {
        stringResource(net.kdhmedia.mochil.R.string.tv_playback_position_unknown,
            formatNowPlayingTime(timeline.positionMs))
    }
    Column {
        Slider(
            value = displayed.toFloat(),
            onValueChange = { dragged = it },
            onValueChangeFinished = { if (dragged >= 0f) playbackViewModel.seekTo(dragged.toLong()); dragged = -1f },
            valueRange = 0f..duration.toFloat(),
            modifier = Modifier.tvFocusBorder().semantics {
                contentDescription = seekLabel
                stateDescription = seekState
            },
        )
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(formatNowPlayingTime(displayed), color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(formatNowPlayingTime(timeline.durationMs), color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

internal fun closeTvNowPlaying(closePlayback: () -> Unit, onBack: () -> Unit) {
    closePlayback()
    onBack()
}

internal fun formatNowPlayingTime(milliseconds: Long): String {
    val seconds = milliseconds.coerceAtLeast(0L) / 1_000L
    val hours = seconds / 3_600L
    val minutes = seconds % 3_600L / 60L
    val remainder = seconds % 60L
    return if (hours > 0) "%d:%02d:%02d".format(hours, minutes, remainder) else "%d:%02d".format(minutes, remainder)
}
