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
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.InsertDriveFile
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.Folder
import androidx.compose.material.icons.outlined.Refresh
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import net.kdhmedia.mochil.webdav.WebDavConnection
import net.kdhmedia.mochil.webdav.WebDavViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
fun WebDavScreen(onBack: () -> Unit, model: WebDavViewModel = viewModel(), playback: PlaybackViewModel = viewModel()) {
    val state by model.state.collectAsStateWithLifecycle()
    var adding by rememberSaveable { mutableStateOf(false) }
    var searching by rememberSaveable { mutableStateOf(false) }
    var query by rememberSaveable { mutableStateOf("") }
    var removalId by rememberSaveable { mutableStateOf<String?>(null) }
    var editingId by rememberSaveable { mutableStateOf<String?>(null) }
    LaunchedEffect(state.readyAudio?.id) {
        state.readyAudio?.let { track -> playback.play(listOf(track), track, { it.id }); model.consumeReadyAudio() }
    }
    Column(Modifier.fillMaxSize().padding(horizontal = 12.dp, vertical = 8.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { if (!model.back()) onBack() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
            Text("WebDAV", Modifier.weight(1f), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
            if (state.selectedId != null) {
                IconButton(onClick = model::indexSelected, enabled = !state.indexing) { Icon(Icons.Outlined.Refresh, "Index WebDAV library") }
                IconButton(onClick = { searching = !searching; if (!searching) query = "" }, enabled = state.indexedEntries.isNotEmpty()) { Icon(Icons.Outlined.Search, "Search indexed library") }
            }
            IconButton(onClick = { adding = true }) { Icon(Icons.Default.Add, "Add WebDAV connection") }
        }
        if (state.connections.isNotEmpty()) Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            LazyRow(Modifier.weight(1f), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                items(state.connections, key = WebDavConnection::id) { value -> FilterChip(state.selectedId == value.id, { model.select(value.id) }, label = { Text(value.name) }) }
            }
            state.connections.firstOrNull { it.id == state.selectedId }?.let { selected ->
                IconButton(onClick = { editingId = selected.id }) { Icon(Icons.Outlined.Edit, "Edit ${selected.name}") }
                IconButton(onClick = { removalId = selected.id }) { Icon(Icons.Outlined.Delete, "Remove ${selected.name}") }
            }
        }
        if (searching) OutlinedTextField(
            query, { query = it.take(200) }, label = { Text("Search indexed library") }, singleLine = true,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 6.dp),
        )
        Text("Read-only remote library. Opening media downloads a private local copy; source files are never changed.", Modifier.padding(horizontal = 8.dp, vertical = 6.dp), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        if (state.indexing) Text("Indexing quietly · ${state.indexedFolders} folders · ${state.indexedMedia} files", Modifier.padding(horizontal = 8.dp, vertical = 4.dp).semantics { liveRegion = LiveRegionMode.Polite }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        state.message?.let { Text(it, Modifier.padding(8.dp).semantics { liveRegion = LiveRegionMode.Polite }, color = if (state.messageIsError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodySmall) }
        val visibleEntries = if (query.isBlank()) state.entries else state.indexedEntries.filter { it.name.contains(query.trim(), ignoreCase = true) }
        LazyColumn(Modifier.weight(1f).fillMaxWidth()) {
            if (state.connections.isEmpty()) item { Column(Modifier.fillParentMaxHeight().fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Text("No WebDAV libraries yet", style = MaterialTheme.typography.titleLarge)
                Text("Add the HTTPS WebDAV address supplied by your storage provider.", Modifier.padding(top = 8.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            } }
            if (state.loading && state.entries.isEmpty()) item { Row(Modifier.fillMaxWidth().padding(24.dp), horizontalArrangement = Arrangement.Center) { if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(28.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(28.dp), strokeWidth = 2.dp) } }
            if (query.isNotBlank() && visibleEntries.isEmpty()) item { Text("No indexed media matches “${query.trim()}”.", Modifier.padding(16.dp), color = MaterialTheme.colorScheme.onSurfaceVariant) }
            items(visibleEntries, key = { it.href }) { entry -> Row(
                Modifier.fillMaxWidth().clickable(role = Role.Button, onClickLabel = if (entry.collection) "Open ${entry.name}" else "Download and open ${entry.name}") { model.open(entry) }.padding(horizontal = 8.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Icon(if (entry.collection) Icons.Outlined.Folder else Icons.AutoMirrored.Outlined.InsertDriveFile, null, tint = MaterialTheme.colorScheme.primary)
                Column(Modifier.weight(1f)) {
                    Text(entry.name, style = MaterialTheme.typography.titleMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    Text(if (entry.collection) "Folder" else listOf(entry.extension.uppercase(), formatByteSize(entry.contentLength)).filter(String::isNotBlank).joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                if (state.openingHref == entry.href) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(22.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp)
            } }
        }
    }
    if (adding) AddWebDavDialog({ adding = false }) { name, url, username, password -> model.add(name, url, username, password); adding = false }
    editingId?.let { id -> state.connections.firstOrNull { it.id == id }?.let { connection ->
        WebDavConnectionDialog(connection, { editingId = null }) { name, url, username, password ->
            model.update(connection, name, url, username, password); editingId = null
        }
    } }
    removalId?.let { id -> state.connections.firstOrNull { it.id == id }?.let { connection -> AlertDialog(
        onDismissRequest = { removalId = null }, title = { Text("Remove ${connection.name}?") },
        text = { Text("The encrypted address and credentials will be removed. Downloaded media remains available.") },
        confirmButton = { TextButton(onClick = { model.remove(connection); removalId = null }) { Text("Remove connection") } },
        dismissButton = { TextButton(onClick = { removalId = null }) { Text("Keep connection") } },
    ) } }
}

@Composable
private fun AddWebDavDialog(onDismiss: () -> Unit, onAdd: (String, String, String, String) -> Unit) {
    WebDavConnectionDialog(null, onDismiss, onAdd)
}

@Composable
private fun WebDavConnectionDialog(existing: WebDavConnection?, onDismiss: () -> Unit, onSave: (String, String, String, String) -> Unit) {
    var name by rememberSaveable(existing?.id) { mutableStateOf(existing?.name.orEmpty()) }; var url by rememberSaveable(existing?.id) { mutableStateOf(existing?.rootUrl.orEmpty()) }
    var username by rememberSaveable(existing?.id) { mutableStateOf(existing?.username.orEmpty()) }; var password by rememberSaveable(existing?.id) { mutableStateOf(existing?.password.orEmpty()) }
    AlertDialog(onDismissRequest = onDismiss, title = { Text(if (existing == null) "Add WebDAV library" else "Edit WebDAV library") }, text = { Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlinedTextField(name, { name = it.take(120) }, label = { Text("Library name") }, singleLine = true)
        OutlinedTextField(url, { url = it.take(4096) }, label = { Text("HTTPS WebDAV address") }, placeholder = { Text("https://cloud.example/remote.php/dav/files/name/") }, singleLine = true)
        OutlinedTextField(username, { username = it.take(320) }, label = { Text("Username (optional)") }, singleLine = true)
        OutlinedTextField(password, { password = it.take(2048) }, label = { Text("Password or app password (optional)") }, singleLine = true, visualTransformation = PasswordVisualTransformation())
        Text("Credentials are encrypted on this device. Mochil requests read access only and never modifies remote files.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    } }, confirmButton = { TextButton(onClick = { onSave(name, url, username, password) }, enabled = name.isNotBlank() && url.isNotBlank() && (username.isBlank() == password.isBlank())) { Text(if (existing == null) "Add library" else "Save changes") } }, dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } })
}

private fun formatFileSize(bytes: Long): String = formatByteSize(bytes)
