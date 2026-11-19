package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDownward
import androidx.compose.material.icons.filled.ArrowUpward
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.automirrored.filled.PlaylistPlay
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicQueueState
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
fun MusicQueuePanel(
    state: MusicQueueState,
    onPlay: (AudioTrack) -> Unit,
    onMove: (Int, Int) -> Unit,
    onRemove: (Int) -> Unit,
    onClear: () -> Unit,
    onOpenPlaylists: () -> Unit,
    onRetry: () -> Unit,
    onBack: () -> Unit,
) {
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(start = 20.dp, end = 8.dp, top = 12.dp, bottom = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to player") }
            Column(Modifier.weight(1f)) {
                Text("Queue", style = MaterialTheme.typography.headlineSmall)
                Text("${state.tracks.size} ${if (state.tracks.size == 1) "track" else "tracks"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (state.saving) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.padding(10.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.padding(10.dp), strokeWidth = 2.dp)
            IconButton(onClick = onOpenPlaylists) { Icon(Icons.AutoMirrored.Filled.PlaylistPlay, contentDescription = "Playlists") }
            if (state.tracks.isNotEmpty()) TextButton(onClick = onClear) { Text("Clear") }
        }
        state.error?.let {
            Column(Modifier.padding(horizontal = 20.dp, vertical = 6.dp).semantics { liveRegion = LiveRegionMode.Polite }) {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                TextButton(onClick = onRetry, enabled = !state.loading && !state.saving) {
                    Text(if (state.loaded) "Retry queue sync" else "Retry loading queue")
                }
            }
        }
        if (state.loading && state.tracks.isEmpty()) {
            Text("Loading queue…", Modifier.padding(20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        if (state.loaded && !state.loading && state.error == null && state.tracks.isEmpty()) {
            Text("Your queue is empty.", Modifier.padding(20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            return@Column
        }
        LazyColumn(Modifier.fillMaxSize()) {
            itemsIndexed(state.tracks, key = { index, track -> "$index:${track.libraryId}:${track.id}" }) { index, track ->
                Row(
                    Modifier.fillMaxWidth().clickable(onClickLabel = "Play ${track.title}", role = Role.Button) { onPlay(track) }.padding(start = 20.dp, end = 4.dp, top = 8.dp, bottom = 8.dp),
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    Column(Modifier.weight(1f)) {
                        Text(track.title, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        Text(track.artist.ifBlank { "Unknown artist" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    IconButton(onClick = { onMove(index, -1) }, enabled = index > 0) { Icon(Icons.Default.ArrowUpward, contentDescription = "Move ${track.title} up") }
                    IconButton(onClick = { onMove(index, 1) }, enabled = index < state.tracks.lastIndex) { Icon(Icons.Default.ArrowDownward, contentDescription = "Move ${track.title} down") }
                    IconButton(onClick = { onRemove(index) }) { Icon(Icons.Default.Close, contentDescription = "Remove ${track.title}") }
                }
            }
        }
    }
}
