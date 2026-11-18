package net.kdhmedia.mochil.ui

import androidx.compose.foundation.focusGroup
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.automirrored.filled.QueueMusic
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.focusRestorer
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.playback.PlaybackViewModel

@Composable
internal fun BoxScope.TvPlaybackDock(
    playbackViewModel: PlaybackViewModel,
    artworkUrl: (net.kdhmedia.mochil.audio.AudioTrack) -> String,
    authorizationHeader: String?,
    onVisibilityChanged: (Boolean) -> Unit = {},
    onExpand: () -> Unit,
) {
    val dock by playbackViewModel.dockState.collectAsStateWithLifecycle()
    val track = dock.current
    LaunchedEffect(track != null) { onVisibilityChanged(track != null) }
    track ?: return
    val shape = RoundedCornerShape(14.dp)
    Surface(
        modifier = Modifier.align(Alignment.BottomEnd).padding(end = 44.dp, bottom = 12.dp).width(480.dp)
            .focusRestorer().focusGroup(),
        shape = shape,
        color = androidx.compose.ui.graphics.Color.Transparent,
    ) {
        Row(
            Modifier.fillMaxWidth().tvGlass(shape, .86f).padding(horizontal = 12.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            AuthenticatedArtwork(
                artworkUrl(track),
                authorizationHeader,
                track.title,
                Modifier.size(52.dp),
            )
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
                Text(track.title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        track.artist.ifBlank { if (track.libraryKind == "podcasts") "Podcast" else "Unknown artist" },
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    TvPlaybackDockClock(playbackViewModel)
                }
            }
            TvIconButton(if (dock.isPlaying) Icons.Default.Pause else Icons.Default.PlayArrow, if (dock.isPlaying) "Pause" else "Play", playbackViewModel::toggle)
            TvIconButton(Icons.AutoMirrored.Filled.QueueMusic, "Open Now Playing", onExpand)
            TvIconButton(Icons.Default.Close, "Close player", playbackViewModel::closePlayback)
        }
    }
}

@Composable
private fun TvPlaybackDockClock(playbackViewModel: PlaybackViewModel) {
    val timeline by playbackViewModel.timelineState.collectAsStateWithLifecycle()
    Text(
        "${formatTvTime(timeline.positionMs)} / ${formatTvTime(timeline.durationMs)}",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        maxLines = 1,
    )
}

private fun formatTvTime(milliseconds: Long): String {
    val seconds = (milliseconds.coerceAtLeast(0) / 1_000)
    val hours = seconds / 3_600
    val minutes = (seconds % 3_600) / 60
    val remainder = seconds % 60
    return if (hours > 0) "%d:%02d:%02d".format(hours, minutes, remainder) else "%d:%02d".format(minutes, remainder)
}
