package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.clickable
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.StateFlow
import net.kdhmedia.mochil.playback.PlaybackDockState
import net.kdhmedia.mochil.playback.PlaybackTimelineState
import net.kdhmedia.mochil.ui.theme.LocalEInkPalette

@Composable
fun MiniPlayer(state: PlaybackDockState, timeline: StateFlow<PlaybackTimelineState>, onToggle: () -> Unit, onClose: () -> Unit, onExpand: () -> Unit, navigationInset: Boolean = true) {
    val track = state.current ?: return
    // No shadow on e-ink: a raised surface over a panel of one grey is a smudge, not a layer.
    Surface(
        modifier = (if (navigationInset) Modifier.navigationBarsPadding() else Modifier).clickable(onClickLabel = "Open player", role = Role.Button, onClick = onExpand),
        tonalElevation = if (LocalEInkPalette.current) 0.dp else 3.dp,
    ) {
        Column {
            MobileMiniPlayerProgress(timeline)
            Row(
                Modifier.fillMaxWidth().padding(start = 20.dp, end = 8.dp, top = 6.dp, bottom = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Column(Modifier.weight(1f)) {
                    Text(track.title, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            track.artist.ifBlank { "Unknown artist" },
                            modifier = Modifier.weight(1f),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        MobileMiniPlayerClock(timeline)
                    }
                }
                IconButton(onClick = onToggle, modifier = Modifier.size(48.dp)) {
                    Icon(if (state.isPlaying) Icons.Default.Pause else Icons.Default.PlayArrow, contentDescription = if (state.isPlaying) "Pause" else "Play")
                }
                IconButton(onClick = onClose, modifier = Modifier.size(48.dp)) {
                    Icon(Icons.Default.Close, contentDescription = "Close player")
                }
            }
        }
    }
}

@Composable
private fun MobileMiniPlayerProgress(timeline: StateFlow<PlaybackTimelineState>) {
    val state by timeline.collectAsStateWithLifecycle()
    LinearProgressIndicator(
        progress = { if (state.durationMs > 0) state.positionMs.toFloat() / state.durationMs else 0f },
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
private fun MobileMiniPlayerClock(timeline: StateFlow<PlaybackTimelineState>) {
    val state by timeline.collectAsStateWithLifecycle()
    Text(
        "  ·  ${playbackClock(state.positionMs)} / ${playbackClock(state.durationMs)}",
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        maxLines = 1,
    )
}
