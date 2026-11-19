package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.DeleteOutline
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.CancellationException
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicPlaylistOverview
import net.kdhmedia.mochil.audio.MusicPlaylistState
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

/**
 * The key a playlist row is drawn under: the server's own identity, the name exactly as stored. Lowercasing it made
 * "Mix" and "mix" the same key, and two playlists with one key crash the lazy list.
 */
internal fun musicPlaylistItemKey(playlist: MusicPlaylistOverview): String = playlist.name

@Composable
fun MusicPlaylistsPanel(
    state: MusicPlaylistState,
    queue: List<AudioTrack>,
    onPlay: suspend (String) -> Unit,
    onSaveQueue: (String, List<AudioTrack>) -> Unit,
    onDelete: (String) -> Unit,
    onBack: () -> Unit,
) {
    var adding by remember { mutableStateOf(false) }
    var name by remember { mutableStateOf("") }
    var playing by remember { mutableStateOf<String?>(null) }
    var playError by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    Column(Modifier.fillMaxSize()) {
        playError?.let { Text(it, Modifier.padding(horizontal = 20.dp), color = MaterialTheme.colorScheme.error) }
        Row(Modifier.fillMaxWidth().padding(start = 8.dp, end = 12.dp, top = 12.dp, bottom = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to queue") }
            Column(Modifier.weight(1f)) {
                Text("Playlists", style = MaterialTheme.typography.headlineSmall)
                Text("Saved on your server", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (state.saving) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.padding(10.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.padding(10.dp), strokeWidth = 2.dp)
            TextButton(onClick = { adding = !adding }, enabled = queue.isNotEmpty() && !state.saving) { Text(if (adding) "Cancel" else "Save queue") }
        }
        if (adding) {
            Row(
                Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                OutlinedTextField(
                    value = name,
                    onValueChange = { if (it.length <= 80) name = it },
                    modifier = Modifier.weight(1f),
                    singleLine = true,
                    label = { Text("Playlist name") },
                )
                Button(onClick = { onSaveQueue(name, queue); name = ""; adding = false }, enabled = name.isNotBlank() && !state.saving) { Text("Save") }
            }
        }
        state.error?.let { Text(it, Modifier.padding(horizontal = 20.dp, vertical = 6.dp), color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
        when {
            state.loading -> if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.align(Alignment.CenterHorizontally).padding(32.dp)) else CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally).padding(32.dp))
            state.playlists.isEmpty() -> Text("No playlists yet. Save the current queue when it is ready.", Modifier.padding(20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            else -> LazyColumn(Modifier.fillMaxSize()) {
                items(state.playlists, key = ::musicPlaylistItemKey) { playlist ->
                    Row(
                        Modifier.fillMaxWidth().clickable(enabled = playlist.trackCount > 0 && playing == null, onClickLabel = "Play ${playlist.name}", role = Role.Button) {
                            playing = playlist.name
                            playError = null
                            scope.launch {
                                try { onPlay(playlist.name) }
                                catch (cancelled: CancellationException) { throw cancelled }
                                catch (_: Exception) { playError = "Could not load this playlist. Try again." }
                                finally { playing = null }
                            }
                        }.padding(start = 20.dp, end = 4.dp, top = 10.dp, bottom = 10.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        if (playing == playlist.name) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(24.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 2.dp)
                        else Icon(Icons.Default.PlayArrow, contentDescription = null, tint = if (playlist.trackCount == 0) MaterialTheme.colorScheme.outline else MaterialTheme.colorScheme.primary)
                        Column(Modifier.weight(1f)) {
                            Text(playlist.name, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                            Text("${playlist.trackCount} ${if (playlist.trackCount == 1) "track" else "tracks"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        IconButton(onClick = { onDelete(playlist.name) }, enabled = !state.saving) { Icon(Icons.Default.DeleteOutline, contentDescription = "Delete ${playlist.name}") }
                    }
                }
            }
        }
    }
}
