package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.podcast.PodcastDiscoveryResult
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

private enum class SubscribeMode { Search, Rss }

@Composable
internal fun PodcastSubscribeDialog(
    state: PodcastState,
    authorizationHeader: String?,
    artworkUrl: (PodcastDiscoveryResult) -> String,
    dismiss: () -> Unit,
    discover: (String) -> Unit,
    subscribe: (String) -> Unit,
) {
    var mode by remember { mutableStateOf(SubscribeMode.Search) }
    var value by remember { mutableStateOf(state.discoveryQuery) }
    val validUrl = value.trim().let { it.startsWith("https://") || it.startsWith("http://") }
    AlertDialog(
        onDismissRequest = { if (!state.subscriptionBusy) dismiss() },
        title = { Text("Add a podcast") },
        text = {
            Column(Modifier.fillMaxWidth().heightIn(max = 480.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                SecondaryTabRow(selectedTabIndex = mode.ordinal) {
                    SubscribeMode.entries.forEach { item -> Tab(selected = mode == item, onClick = { mode = item; value = "" }, text = { Text(if (item == SubscribeMode.Search) "Search" else "RSS address") }) }
                }
                OutlinedTextField(
                    value = value, onValueChange = { value = it }, singleLine = true, enabled = !state.subscriptionBusy,
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(if (mode == SubscribeMode.Search) "Podcast name" else "RSS feed address") },
                    leadingIcon = if (mode == SubscribeMode.Search) {{ Icon(Icons.Default.Search, contentDescription = null) }} else null,
                    keyboardOptions = KeyboardOptions(imeAction = if (mode == SubscribeMode.Search) ImeAction.Search else ImeAction.Done),
                    keyboardActions = KeyboardActions(onSearch = { discover(value) }, onDone = { if (validUrl) subscribe(value) }),
                )
                if (mode == SubscribeMode.Search) DiscoveryResults(state, authorizationHeader, artworkUrl, subscribe)
                else Text("Use the direct RSS or Atom feed address supplied by the publisher.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        },
        confirmButton = {
            if (mode == SubscribeMode.Search) TextButton(enabled = value.isNotBlank() && !state.discoverySearching && !state.subscriptionBusy, onClick = { discover(value) }) { Text(if (state.discoverySearching) "Searching…" else "Search") }
            else TextButton(enabled = validUrl && !state.subscriptionBusy, onClick = { subscribe(value) }) { Text(if (state.subscriptionBusy) "Adding…" else "Subscribe") }
        },
        dismissButton = { TextButton(enabled = !state.subscriptionBusy, onClick = dismiss) { Text("Cancel") } },
    )
}

@Composable
private fun DiscoveryResults(state: PodcastState, authorizationHeader: String?, artworkUrl: (PodcastDiscoveryResult) -> String, subscribe: (String) -> Unit) {
    when {
        state.discoverySearching -> Box(Modifier.fillMaxWidth().height(96.dp), contentAlignment = Alignment.Center) { if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(28.dp)) else CircularProgressIndicator(Modifier.size(28.dp)) }
        state.discoveryError != null -> Text(state.discoveryError, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
        state.discoveryQuery.isNotBlank() && state.discoveryResults.isEmpty() -> Text("No podcasts found.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            items(state.discoveryResults, key = PodcastDiscoveryResult::feedUrl) { result ->
                Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    AuthenticatedArtwork(artworkUrl(result), authorizationHeader, result.title, Modifier.size(48.dp))
                    Column(Modifier.weight(1f)) {
                        Text(result.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        val detail = listOf(result.author, result.genre, result.episodeCount.takeIf { it > 0 }?.let { "$it episodes" }.orEmpty()).filter(String::isNotBlank).joinToString(" · ")
                        if (detail.isNotBlank()) Text(detail, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    TextButton(enabled = !state.subscriptionBusy, onClick = { subscribe(result.feedUrl) }) { Text("Add") }
                }
            }
        }
    }
}
