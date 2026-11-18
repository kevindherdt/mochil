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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Storage
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.local.LocalFolderSource
import net.kdhmedia.mochil.local.LocalMediaItem
import net.kdhmedia.mochil.local.LocalMediaKind
import net.kdhmedia.mochil.local.LocalSourceState
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
fun LocalMediaLibrary(
    state: LocalSourceState,
    selectedKind: LocalMediaKind,
    onKind: (LocalMediaKind) -> Unit,
    onOpen: (LocalMediaItem) -> Unit,
    onRefresh: (LocalFolderSource) -> Unit,
    onRemove: (LocalFolderSource) -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyColumn(modifier.fillMaxSize(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        if (state.folders.isEmpty()) item {
            Column(Modifier.fillParentMaxHeight().fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Icon(Icons.Default.Storage, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                Text("No local libraries yet", Modifier.padding(top = 14.dp), style = MaterialTheme.typography.titleLarge)
                Text("Choose a media folder or connect a server. Both remain independent sources.", Modifier.padding(top = 8.dp), style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        } else {
            item { Text("On this device", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold) }
            items(state.folders, key = LocalFolderSource::uri) { folder -> FolderSourceRow(folder, onRefresh, onRemove) }
            item {
                LazyRow(Modifier.fillMaxWidth().padding(top = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(LocalMediaKind.entries, key = LocalMediaKind::name) { kind ->
                        val count = state.items.count { it.kind == kind }
                        FilterChip(selected = selectedKind == kind, onClick = { onKind(kind) }, label = { Text("${kind.label} · $count") })
                    }
                }
            }
            val visible = state.items.filter { it.kind == selectedKind }
            if (visible.isEmpty() && state.folders.none(LocalFolderSource::scanning)) item {
                Text("No ${selectedKind.label.lowercase()} found in these folders.", Modifier.padding(vertical = 20.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            items(visible, key = LocalMediaItem::documentUri) { item ->
                Row(
                    Modifier.fillMaxWidth().clickable(role = Role.Button, onClickLabel = "Open ${item.title}") { onOpen(item) }.padding(vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(14.dp),
                ) {
                    Text(item.extension.uppercase(), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
                    Column(Modifier.weight(1f)) {
                        Text(item.title, style = MaterialTheme.typography.titleMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        Text(listOfNotNull(item.artist.takeIf(String::isNotBlank), item.album.takeIf(String::isNotBlank), item.name.takeIf { it != item.title }).joinToString(" · ").ifBlank { item.extension.uppercase() }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
        }
    }
}

@Composable
private fun FolderSourceRow(folder: LocalFolderSource, onRefresh: (LocalFolderSource) -> Unit, onRemove: (LocalFolderSource) -> Unit) {
    Surface(Modifier.fillMaxWidth(), shape = MaterialTheme.shapes.medium, color = MaterialTheme.colorScheme.surfaceVariant) {
        Row(Modifier.fillMaxWidth().padding(start = 16.dp, top = 8.dp, bottom = 8.dp, end = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(folder.name, style = MaterialTheme.typography.titleMedium)
                Text(if (folder.scanning) "Scanning…" else folder.error ?: "${folder.mediaCount} supported ${if (folder.mediaCount == 1) "file" else "files"}", style = MaterialTheme.typography.bodySmall, color = if (folder.error == null) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.error)
            }
            if (folder.scanning) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(24.dp).padding(2.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(24.dp).padding(2.dp), strokeWidth = 2.dp)
            else IconButton(onClick = { onRefresh(folder) }) { Icon(Icons.Default.Refresh, contentDescription = "Rescan ${folder.name}") }
            IconButton(onClick = { onRemove(folder) }) { Icon(Icons.Default.Delete, contentDescription = "Remove ${folder.name}") }
        }
    }
}

private val LocalMediaKind.label get() = when (this) {
    LocalMediaKind.Publications -> "Books"
    LocalMediaKind.Audio -> "Audio"
    LocalMediaKind.Video -> "Video"
}
