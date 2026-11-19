package net.kdhmedia.mochil.ui

import androidx.activity.compose.BackHandler
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.QueueMusic
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.audio.MusicPlaylistState

data class MusicPlaylistBindings(
    val state: MusicPlaylistState,
    val select: (String?) -> Unit,
    val play: (List<AudioTrack>, AudioTrack) -> Unit,
    val move: (String, Int, Int) -> Unit,
    val remove: (String, Int) -> Unit,
    val rename: (String, String) -> Unit,
    val delete: (String) -> Unit,
    val add: (AudioTrack, String) -> Unit,
    val offline: PlaylistOfflineBindings,
    val artworkUrl: (String) -> String,
)

data class PlaylistOfflineBindings(
    val downloadedIds: Set<String>,
    val busy: Boolean,
    val completed: Int,
    val total: Int,
    val download: (List<AudioTrack>) -> Unit,
    val remove: (List<AudioTrack>) -> Unit,
)

@Composable
internal fun MusicPlaylistLibrary(bindings: MusicPlaylistBindings, artworkUrl: (AudioTrack) -> String, header: String?) {
    val state = bindings.state
    state.error?.let { Text(it, Modifier.padding(horizontal = 20.dp, vertical = 6.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error) }
    state.notice?.let { Text(it, Modifier.padding(horizontal = 20.dp, vertical = 6.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary) }
    val selected = state.selectedPlaylist
    if (state.selectedName == null) {
        when {
            state.loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }) else CircularProgressIndicator()
            }
            state.playlists.isEmpty() -> Box(Modifier.fillMaxSize().padding(32.dp), contentAlignment = Alignment.Center) { Text("No playlists yet. Add a track from its menu to create one.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            else -> LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(vertical = 8.dp)) {
                items(state.playlists, key = { it.name.lowercase() }) { playlist ->
                    Row(
                        Modifier.fillMaxWidth().clickable(onClickLabel = "Open ${playlist.name}", role = Role.Button) { bindings.select(playlist.name) }.padding(horizontal = 20.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically,
                    ) {
                        PlaylistArtworkURLs(playlist.artworkUrls.map(bindings.artworkUrl), header, playlist.name)
                        Column(Modifier.weight(1f)) {
                            Text(playlist.name, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            Text("${playlist.trackCount} ${if (playlist.trackCount == 1) "track" else "tracks"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                }
            }
        }
        return
    }
    BackHandler { bindings.select(null) }
    if (selected == null) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { bindings.select(null) }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back to playlists") }
            if (state.detailLoading) {
                if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(24.dp)) else CircularProgressIndicator(Modifier.size(24.dp))
            }
            else TextButton(onClick = { bindings.select(state.selectedName) }) { Text("Retry") }
        }
        return
    }
    PlaylistDetail(selected.name, selected.tracks, state.saving, artworkUrl, header, bindings)
}

@Composable
private fun PlaylistArtworkURLs(covers: List<String>, header: String?, title: String) {
    if (covers.isEmpty()) return Box(Modifier.size(56.dp), contentAlignment = Alignment.Center) { Icon(Icons.AutoMirrored.Filled.QueueMusic, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant) }
    if (covers.size < 4) return AuthenticatedArtwork(covers.first(), header, title, Modifier.size(56.dp))
    Column(Modifier.size(56.dp), verticalArrangement = Arrangement.spacedBy(1.dp)) {
        for (row in 0..1) Row(Modifier.weight(1f), horizontalArrangement = Arrangement.spacedBy(1.dp)) {
            for (column in 0..1) {
                val track = covers[row * 2 + column]
                AuthenticatedArtwork(track, header, title, Modifier.weight(1f).fillMaxHeight(), cornerRadius = 3.dp)
            }
        }
    }
}

@Composable
private fun PlaylistDetail(name: String, tracks: List<AudioTrack>, saving: Boolean, artworkUrl: (AudioTrack) -> String, header: String?, bindings: MusicPlaylistBindings) {
    var menu by remember { mutableStateOf(false) }
    var renaming by remember { mutableStateOf(false) }
    var deleting by remember { mutableStateOf(false) }
    var removingDownloads by remember { mutableStateOf(false) }
    val offline = playlistOfflineState(tracks, bindings.offline.downloadedIds)
    val downloading = bindings.offline.busy
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { bindings.select(null) }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to playlists") }
            Column(Modifier.weight(1f)) {
                Text(name, style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(playlistOfflineSummary(tracks, offline), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (saving) {
                if (LocalReduceMotion.current) {
                    CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(22.dp), strokeWidth = 2.dp)
                } else {
                    CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp)
                }
            }
            if (bindings.offline.total > 0) Text("${bindings.offline.completed}/${bindings.offline.total}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            IconButton(onClick = { tracks.firstOrNull()?.let { bindings.play(tracks, it) } }, enabled = tracks.isNotEmpty()) { Icon(Icons.Default.PlayArrow, contentDescription = "Play playlist") }
            Box {
                IconButton(onClick = { menu = true }, enabled = !downloading) {
                    if (downloading) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(24.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 2.dp)
                    else Icon(Icons.Default.MoreVert, contentDescription = "Playlist options")
                }
                DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                    if (offline.downloaded < offline.total) DropdownMenuItem(
                        text = { Text("Download playlist") },
                        leadingIcon = { Icon(Icons.Default.CloudDownload, null) },
                        onClick = { menu = false; bindings.offline.download(tracks) },
                    )
                    if (offline.downloaded > 0) DropdownMenuItem(
                        text = { Text("Remove offline downloads") },
                        leadingIcon = { Icon(Icons.Default.DeleteSweep, null) },
                        onClick = { menu = false; removingDownloads = true },
                    )
                    if (tracks.isNotEmpty()) HorizontalDivider()
                    DropdownMenuItem(text = { Text("Rename") }, leadingIcon = { Icon(Icons.Default.Edit, null) }, onClick = { menu = false; renaming = true })
                    DropdownMenuItem(text = { Text("Delete") }, leadingIcon = { Icon(Icons.Default.DeleteOutline, null) }, onClick = { menu = false; deleting = true })
                }
            }
        }
        LazyColumn(Modifier.fillMaxSize()) {
            itemsIndexed(tracks, key = { index, track -> "$index:${track.libraryId}:${track.id}" }) { index, track ->
                PlaylistTrackRow(track, artworkUrl(track), header, index, tracks.lastIndex, saving, { bindings.play(tracks, track) }, { direction -> bindings.move(name, index, direction) }, { bindings.remove(name, index) })
            }
        }
    }
    if (renaming) RenamePlaylistDialog(name, { renaming = false }) { newName -> renaming = false; bindings.rename(name, newName) }
    if (removingDownloads) AlertDialog(
        onDismissRequest = { removingDownloads = false },
        title = { Text("Remove offline downloads?") },
        text = { Text("Remove ${offline.downloaded} downloaded ${if (offline.downloaded == 1) "track" else "tracks"} from this device? The playlist will stay available for streaming.") },
        confirmButton = { TextButton(onClick = { removingDownloads = false; bindings.offline.remove(tracks) }) { Text("Remove") } },
        dismissButton = { TextButton(onClick = { removingDownloads = false }) { Text("Cancel") } },
    )
    if (deleting) AlertDialog(
        onDismissRequest = { deleting = false },
        title = { Text("Delete playlist?") },
        text = { Text("Delete “$name”? Your music files will not be removed.") },
        confirmButton = { TextButton(onClick = { deleting = false; bindings.select(null); bindings.delete(name) }) { Text("Delete") } },
        dismissButton = { TextButton(onClick = { deleting = false }) { Text("Cancel") } },
    )
}

data class PlaylistOfflineState(val downloaded: Int, val total: Int)

internal fun playlistOfflineState(tracks: List<AudioTrack>, downloadedIds: Set<String>): PlaylistOfflineState {
    val unique = tracks.distinctBy(AudioTrack::identityKey)
    return PlaylistOfflineState(unique.count { it.identityKey in downloadedIds }, unique.size)
}

internal fun playlistOfflineSummary(tracks: List<AudioTrack>, state: PlaylistOfflineState): String {
    val trackLabel = "${state.total} ${if (state.total == 1) "track" else "tracks"}"
    val size = tracks.distinctBy(AudioTrack::identityKey).sumOf { it.sizeBytes.coerceAtLeast(0) }
    val parts = mutableListOf(trackLabel)
    if (state.downloaded > 0) parts += "${state.downloaded} offline"
    if (size > 0) parts += formatByteSize(size)
    return parts.joinToString(" · ")
}

@Composable
private fun PlaylistTrackRow(track: AudioTrack, artwork: String, header: String?, index: Int, lastIndex: Int, saving: Boolean, onPlay: () -> Unit, onMove: (Int) -> Unit, onRemove: () -> Unit) {
    var menu by remember { mutableStateOf(false) }
    Box {
        Row(
            Modifier.fillMaxWidth().combinedClickable(
                role = Role.Button,
                onClickLabel = "Play ${track.title}",
                onLongClickLabel = "Playlist track options",
                onClick = onPlay,
                onLongClick = { if (!saving) menu = true },
            ).semantics {
                customActions = listOf(CustomAccessibilityAction("Playlist track options") {
                    if (!saving) menu = true
                    !saving
                })
            }.padding(horizontal = 20.dp, vertical = 7.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            AuthenticatedArtwork(artwork, header, track.title, Modifier.size(48.dp))
            Column(Modifier.weight(1f)) {
                Text(track.title, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(track.artist.ifBlank { "Unknown artist" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            Text(formatTrackDuration(track.durationSeconds), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (saving) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(20.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
        }
            DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                DropdownMenuItem(text = { Text("Move up") }, enabled = index > 0, onClick = { menu = false; onMove(-1) })
                DropdownMenuItem(text = { Text("Move down") }, enabled = index < lastIndex, onClick = { menu = false; onMove(1) })
                DropdownMenuItem(text = { Text("Remove from playlist") }, onClick = { menu = false; onRemove() })
            }
    }
}

@Composable
private fun RenamePlaylistDialog(current: String, onDismiss: () -> Unit, onRename: (String) -> Unit) {
    var name by remember(current) { mutableStateOf(current) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Rename playlist") },
        text = { OutlinedTextField(name, { name = it.take(80) }, singleLine = true, label = { Text("Name") }) },
        confirmButton = { TextButton(onClick = { onRename(name.trim()) }, enabled = name.isNotBlank() && name.trim() != current) { Text("Rename") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
