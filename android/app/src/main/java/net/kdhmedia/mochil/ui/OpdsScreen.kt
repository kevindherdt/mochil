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
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.Folder
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.automirrored.outlined.InsertDriveFile
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
import net.kdhmedia.mochil.opds.OpdsConnection
import net.kdhmedia.mochil.opds.OpdsState
import net.kdhmedia.mochil.opds.OpdsViewModel
import net.kdhmedia.mochil.source.SourceItem
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
fun OpdsScreen(onBack: () -> Unit, model: OpdsViewModel = viewModel()) {
    val state by model.state.collectAsStateWithLifecycle()
    var adding by rememberSaveable { mutableStateOf(false) }
    var searching by rememberSaveable { mutableStateOf(false) }
    var query by rememberSaveable { mutableStateOf("") }
    var pendingRemoval by rememberSaveable { mutableStateOf<String?>(null) }
    var editingId by rememberSaveable { mutableStateOf<String?>(null) }
    Column(Modifier.fillMaxSize().padding(horizontal = 12.dp, vertical = 8.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = { if (!model.back()) onBack() }) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") }
            Text(state.pageTitle.ifBlank { "OPDS catalogues" }, Modifier.weight(1f), style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (state.selectedId != null && state.searchAvailable) IconButton(onClick = { searching = !searching }) { Icon(Icons.Outlined.Search, contentDescription = "Search catalogue") }
            IconButton(onClick = { adding = true }) { Icon(Icons.Default.Add, contentDescription = "Add OPDS catalogue") }
        }
        if (state.connections.isNotEmpty()) ConnectionSelector(state, model::select, { editingId = it.id }) { pendingRemoval = it.id }
        if (searching) OutlinedTextField(
            value = query,
            onValueChange = { query = it.take(200) },
            label = { Text("Search catalogue") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 6.dp),
            trailingIcon = { TextButton(onClick = { model.search(query) }, enabled = query.isNotBlank()) { Text("Search") } },
        )
        state.message?.let { Text(it, Modifier.padding(horizontal = 8.dp, vertical = 6.dp).semantics { liveRegion = LiveRegionMode.Polite }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error) }
        OpdsContents(state, model::openFolder, model::open, model::requestArtwork, model::loadMore, Modifier.weight(1f))
    }
    if (adding) AddOpdsDialog(onDismiss = { adding = false }) { name, url, username, password ->
        model.add(name, url, username, password); adding = false
    }
    editingId?.let { id -> state.connections.firstOrNull { it.id == id }?.let { connection ->
        OpdsConnectionDialog(connection, { editingId = null }) { name, url, username, password ->
            model.update(connection, name, url, username, password); editingId = null
        }
    } }
    pendingRemoval?.let { id -> state.connections.firstOrNull { it.id == id }?.let { connection ->
        AlertDialog(
            onDismissRequest = { pendingRemoval = null },
            title = { Text("Remove ${connection.name}?") },
            text = { Text("The saved address and credentials will be removed. Books already downloaded remain available.") },
            confirmButton = { TextButton(onClick = { model.remove(connection); pendingRemoval = null }) { Text("Remove catalogue") } },
            dismissButton = { TextButton(onClick = { pendingRemoval = null }) { Text("Cancel") } },
        )
    } }
}

@Composable
private fun ConnectionSelector(state: OpdsState, onSelect: (String) -> Unit, onEdit: (OpdsConnection) -> Unit, onRemove: (OpdsConnection) -> Unit) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        LazyRow(Modifier.weight(1f), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            items(state.connections, key = OpdsConnection::id) { connection ->
                FilterChip(selected = state.selectedId == connection.id, onClick = { onSelect(connection.id) }, label = { Text(connection.name, maxLines = 1) })
            }
        }
        state.connections.firstOrNull { it.id == state.selectedId }?.let { selected ->
            IconButton(onClick = { onEdit(selected) }) { Icon(Icons.Outlined.Edit, contentDescription = "Edit ${selected.name}") }
            IconButton(onClick = { onRemove(selected) }) { Icon(Icons.Outlined.Delete, contentDescription = "Remove ${selected.name}") }
        }
    }
}

@Composable
private fun OpdsContents(state: OpdsState, onFolder: (SourceItem) -> Unit, onOpen: (SourceItem) -> Unit, onArtwork: (SourceItem) -> Unit, onMore: () -> Unit, modifier: Modifier) {
    LazyColumn(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(2.dp)) {
        if (state.connections.isEmpty()) item {
            Column(Modifier.fillParentMaxHeight().fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Text("No OPDS catalogues yet", style = MaterialTheme.typography.titleLarge)
                Text("Add the HTTPS address provided by your library or self-hosted catalogue.", Modifier.padding(top = 8.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        if (state.loading && state.items.isEmpty()) item { Row(Modifier.fillMaxWidth().padding(24.dp), horizontalArrangement = Arrangement.Center) { if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(28.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(28.dp), strokeWidth = 2.dp) } }
        items(state.items, key = { "${it.id}:${it.navigationUrl}:${it.acquisitionUrl}" }) { item ->
            val folder = item.navigationUrl != null && item.acquisitionUrl == null
            if (!folder) LaunchedEffect(item.artworkUrl) { onArtwork(item) }
            Row(
                Modifier.fillMaxWidth().clickable(role = Role.Button, onClickLabel = if (folder) "Open ${item.title}" else "Download and open ${item.title}") { if (folder) onFolder(item) else onOpen(item) }.padding(horizontal = 8.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                if (folder) Icon(Icons.Outlined.Folder, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                else state.artwork[item.artworkUrl]?.let { artwork -> AuthenticatedArtwork(artwork, null, item.title, Modifier.size(width = 42.dp, height = 58.dp)) }
                    ?: Icon(Icons.AutoMirrored.Outlined.InsertDriveFile, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.size(42.dp))
                Column(Modifier.weight(1f)) {
                    Text(item.title, style = MaterialTheme.typography.titleMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    val detail = if (folder) "Catalogue" else item.authors.joinToString().ifBlank { item.mediaType.substringAfter('/').uppercase() }
                    Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                }
                if (item.id in state.openingIds) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(22.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp)
            }
        }
        if (state.nextUrl != null) item { TextButton(onClick = onMore, enabled = !state.loading, modifier = Modifier.fillMaxWidth()) { Text(if (state.loading) "Loading…" else "Load more") } }
    }
}

@Composable
private fun AddOpdsDialog(onDismiss: () -> Unit, onAdd: (String, String, String, String) -> Unit) {
    OpdsConnectionDialog(null, onDismiss, onAdd)
}

@Composable
private fun OpdsConnectionDialog(existing: OpdsConnection?, onDismiss: () -> Unit, onSave: (String, String, String, String) -> Unit) {
    var name by rememberSaveable(existing?.id) { mutableStateOf(existing?.name.orEmpty()) }; var url by rememberSaveable(existing?.id) { mutableStateOf(existing?.catalogueUrl.orEmpty()) }
    var username by rememberSaveable(existing?.id) { mutableStateOf(existing?.username.orEmpty()) }; var password by rememberSaveable(existing?.id) { mutableStateOf(existing?.password.orEmpty()) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(if (existing == null) "Add OPDS catalogue" else "Edit OPDS catalogue") },
        text = { Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(name, { name = it.take(120) }, label = { Text("Catalogue name") }, singleLine = true)
            OutlinedTextField(url, { url = it.take(2048) }, label = { Text("HTTPS OPDS address") }, placeholder = { Text("https://example.org/opds") }, singleLine = true)
            OutlinedTextField(username, { username = it.take(320) }, label = { Text("Username (optional)") }, singleLine = true)
            OutlinedTextField(password, { password = it.take(2048) }, label = { Text("Password (optional)") }, singleLine = true, visualTransformation = PasswordVisualTransformation())
            Text("Credentials are encrypted on this device and are sent only to this catalogue's exact origin.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        } },
        confirmButton = { TextButton(onClick = { onSave(name, url, username, password) }, enabled = name.isNotBlank() && url.isNotBlank()) { Text(if (existing == null) "Add catalogue" else "Save changes") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
