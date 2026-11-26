package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.ui.kit.LucideIcons
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ViewList
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.podcast.*

data class PodcastBindings(
    val state: PodcastState,
    val showArtworkUrl: (PodcastShow) -> String,
    val episodeArtworkUrl: (AudioTrack) -> String,
    val open: (String) -> Unit,
    val openShow: (PodcastShow) -> Unit,
    val closeShow: () -> Unit,
    val setLayout: (PodcastLayout) -> Unit,
    val search: (String) -> Unit,
    val setPlayedFilter: (PodcastPlayedFilter) -> Unit,
    val setEpisodeSort: (PodcastEpisodeSort) -> Unit,
    val setPlayed: (AudioTrack, Boolean) -> Unit,
    val play: (AudioTrack) -> Unit,
    val toggleDownload: (AudioTrack) -> Unit,
    val loadMore: () -> Unit,
    val retry: () -> Unit,
    val subscribe: (String) -> Unit,
    val unsubscribe: (PodcastShow) -> Unit,
    val discover: (String) -> Unit,
    val clearDiscovery: () -> Unit,
    val discoveryArtworkUrl: (PodcastDiscoveryResult) -> String,
    val refreshShow: (PodcastShow) -> Unit,
    val updatePolicy: (PodcastShow, PodcastPolicy) -> Unit,
    val devicePolicy: (PodcastShow) -> PodcastDevicePolicy,
    val updateDevicePolicy: (PodcastShow, PodcastDevicePolicy) -> Unit,
)

@Composable
fun PodcastScreen(
    state: PodcastState,
    showArtworkUrl: (PodcastShow) -> String,
    episodeArtworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    onOpenShow: (PodcastShow) -> Unit,
    onCloseShow: () -> Unit,
    onLayout: (PodcastLayout) -> Unit,
    onSearch: (String) -> Unit,
    onPlayedFilter: (PodcastPlayedFilter) -> Unit,
    onEpisodeSort: (PodcastEpisodeSort) -> Unit,
    onSetPlayed: (AudioTrack, Boolean) -> Unit,
    onPlay: (AudioTrack) -> Unit,
    onToggleDownload: (AudioTrack) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
    onSubscribe: (String) -> Unit,
    onUnsubscribe: (PodcastShow) -> Unit,
    onDiscover: (String) -> Unit,
    onClearDiscovery: () -> Unit,
    discoveryArtworkUrl: (PodcastDiscoveryResult) -> String,
    onRefreshShow: (PodcastShow) -> Unit,
    onUpdatePolicy: (PodcastShow, PodcastPolicy) -> Unit,
    devicePolicy: (PodcastShow) -> PodcastDevicePolicy,
    onUpdateDevicePolicy: (PodcastShow, PodcastDevicePolicy) -> Unit,
) {
    var searchExpanded by remember { mutableStateOf(false) }
    var subscribeOpen by remember { mutableStateOf(false) }
    var removeShow by remember { mutableStateOf<PodcastShow?>(null) }
    var policyShow by remember { mutableStateOf<PodcastShow?>(null) }
    if (subscribeOpen) PodcastSubscribeDialog(state, authorizationHeader, discoveryArtworkUrl, { subscribeOpen = false; onClearDiscovery() }, onDiscover) { onSubscribe(it); subscribeOpen = false; onClearDiscovery() }
    policyShow?.let { show ->
        val device = remember(show.id) { devicePolicy(show) }
        PodcastPolicyDialog(show, device, state.subscriptionBusy, { policyShow = null }) { server, deviceUpdate -> onUpdatePolicy(show, server); onUpdateDevicePolicy(show, deviceUpdate); policyShow = null }
    }
    removeShow?.let { show ->
        AlertDialog(onDismissRequest = { if (!state.subscriptionBusy) removeShow = null }, title = { Text("Unsubscribe?") },
            text = { Text("Remove ${show.title} and its server-downloaded episodes? Your listening progress is unaffected.") },
            confirmButton = { TextButton(enabled = !state.subscriptionBusy, onClick = { onUnsubscribe(show); removeShow = null }) { Text("Unsubscribe") } },
            dismissButton = { TextButton(enabled = !state.subscriptionBusy, onClick = { removeShow = null }) { Text("Cancel") } })
    }
    BoxWithConstraints(Modifier.fillMaxSize()) {
        val twoPane = usesMediaTwoPane(maxWidth.value.toInt(), state.selectedShow != null)
        if (twoPane) {
            BackHandler(onBack = onCloseShow)
            Row(Modifier.fillMaxSize()) {
                Box(Modifier.weight(.43f)) {
                    PodcastBrowse(state.copy(loading = false, error = null), showArtworkUrl, authorizationHeader, onOpenShow, onLayout, onSearch, onLoadMore, onRetry, { subscribeOpen = true }, { policyShow = it }, onRefreshShow, { removeShow = it }, searchExpanded) { searchExpanded = !searchExpanded }
                }
                VerticalDivider()
                Box(Modifier.weight(.57f)) { PodcastEpisodes(state, state.selectedShow?.let(showArtworkUrl).orEmpty(), authorizationHeader, onCloseShow, onPlayedFilter, onEpisodeSort, onSetPlayed, onPlay, onToggleDownload, onLoadMore, onRetry) }
            }
        } else if (state.selectedShow != null) {
            BackHandler(onBack = onCloseShow)
            PodcastEpisodes(state, showArtworkUrl(state.selectedShow), authorizationHeader, onCloseShow, onPlayedFilter, onEpisodeSort, onSetPlayed, onPlay, onToggleDownload, onLoadMore, onRetry)
        } else {
            PodcastBrowse(state, showArtworkUrl, authorizationHeader, onOpenShow, onLayout, onSearch, onLoadMore, onRetry, { subscribeOpen = true }, { policyShow = it }, onRefreshShow, { removeShow = it }, searchExpanded) { searchExpanded = !searchExpanded }
        }
    }
}

@Composable
private fun PodcastBrowse(
    state: PodcastState,
    showArtworkUrl: (PodcastShow) -> String,
    authorizationHeader: String?,
    onOpenShow: (PodcastShow) -> Unit,
    onLayout: (PodcastLayout) -> Unit,
    onSearch: (String) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
    onSubscribe: () -> Unit,
    onPolicy: (PodcastShow) -> Unit,
    onRefresh: (PodcastShow) -> Unit,
    onRemove: (PodcastShow) -> Unit,
    searchExpanded: Boolean,
    onToggleSearch: () -> Unit,
) {
    Column(Modifier.fillMaxSize()) {
        val covers = state.layout == PodcastLayout.Covers
        HeaderActions(listOf(
            HeaderAction(LucideIcons.Plus, "Subscribe to a podcast", onSubscribe, enabled = !state.subscriptionBusy),
            HeaderAction(LucideIcons.Search, "Search podcast shows", onToggleSearch, active = searchExpanded || state.query.isNotBlank()),
            HeaderAction(if (covers) LucideIcons.List else LucideIcons.LayoutGrid, if (covers) "Use list view" else "Use cover view", { onLayout(if (covers) PodcastLayout.List else PodcastLayout.Covers) }),
        ))
        MotionVisibility(searchExpanded) { PodcastSearch(state.query, onSearch) { if (searchExpanded) onToggleSearch() } }
        state.message?.let { Text(it, Modifier.padding(horizontal = 20.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = if (state.messageIsError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant) }
        if (state.offlineOnly) Row(Modifier.fillMaxWidth().padding(start = 20.dp, end = 12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("Downloaded episodes remain available", Modifier.weight(1f), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
            TextButton(onClick = onRetry) { Text("Retry") }
        }
        when {
            state.loading -> AccessibleLoadingIndicator("Loading podcast shows", Modifier.fillMaxSize())
            state.error != null && state.shows.isEmpty() -> PodcastStatus { Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant); Button(onClick = onRetry) { Text("Try again") } }
            state.shows.isEmpty() -> PodcastStatus { Text(if (state.query.isBlank()) "No podcast shows are subscribed." else "No shows match this search.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            state.layout == PodcastLayout.Covers -> PodcastShowGrid(state, showArtworkUrl, authorizationHeader, onOpenShow, onPolicy, onRefresh, onRemove, onLoadMore, onRetry)
            else -> PodcastShowList(state, showArtworkUrl, authorizationHeader, onOpenShow, onPolicy, onRefresh, onRemove, onLoadMore, onRetry)
        }
    }
}

@Composable
private fun PodcastSearch(query: String, onSearch: (String) -> Unit, onDone: () -> Unit) {
    var draft by remember(query) { mutableStateOf(query) }
    val focus = LocalFocusManager.current
    OutlinedTextField(
        value = draft, onValueChange = { draft = it }, singleLine = true,
        modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
        label = { Text("Search podcast shows") }, leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
        trailingIcon = if (draft.isNotEmpty()) {{ IconButton(onClick = { draft = ""; onSearch("") }) { Icon(Icons.Default.Close, contentDescription = "Clear search") } }} else null,
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = KeyboardActions(onSearch = { onSearch(draft); focus.clearFocus(); onDone() }),
    )
}

@Composable
private fun PodcastShowGrid(state: PodcastState, artworkUrl: (PodcastShow) -> String, header: String?, onOpen: (PodcastShow) -> Unit, onPolicy: (PodcastShow) -> Unit, onRefresh: (PodcastShow) -> Unit, onRemove: (PodcastShow) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    LazyVerticalGrid(columns = GridCells.Adaptive(136.dp), contentPadding = PaddingValues(20.dp), horizontalArrangement = Arrangement.spacedBy(14.dp), verticalArrangement = Arrangement.spacedBy(22.dp)) {
        items(state.shows, key = PodcastShow::id) { show ->
            Column(Modifier.gridCard().clickable(onClickLabel = "Open ${show.title}", role = Role.Button) { onOpen(show) }.padding(10.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                AuthenticatedArtwork(artworkUrl(show), header, show.title, Modifier.fillMaxWidth().aspectRatio(1f))
                Row(verticalAlignment = Alignment.Top) { Text(show.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f)); PodcastShowMenu(show, onPolicy, onRefresh, onRemove) }
                Text(episodeCount(show), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                show.feedNote()?.let { Text(it, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.error) }
            }
        }
        podcastPaging(state, onMore, onRetry)
    }
}

@Composable
private fun PodcastShowList(state: PodcastState, artworkUrl: (PodcastShow) -> String, header: String?, onOpen: (PodcastShow) -> Unit, onPolicy: (PodcastShow) -> Unit, onRefresh: (PodcastShow) -> Unit, onRemove: (PodcastShow) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        items(state.shows, key = PodcastShow::id) { show ->
            Row(Modifier.fillMaxWidth().clickable(onClickLabel = "Open ${show.title}", role = Role.Button) { onOpen(show) }.padding(horizontal = 20.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically) {
                AuthenticatedArtwork(artworkUrl(show), header, show.title, Modifier.size(56.dp))
                Column(Modifier.weight(1f)) {
                    Text(show.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text(show.description.ifBlank { episodeCount(show) }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    show.feedNote()?.let { Text(it, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.error) }
                }
                PodcastShowMenu(show, onPolicy, onRefresh, onRemove)
            }
        }
        podcastPaging(state, onMore, onRetry)
    }
}

@Composable private fun PodcastShowMenu(show: PodcastShow, onPolicy: (PodcastShow) -> Unit, onRefresh: (PodcastShow) -> Unit, onRemove: (PodcastShow) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box { IconButton(onClick = { open = true }) { Icon(Icons.Default.MoreVert, contentDescription = "Options for ${show.title}") }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            DropdownMenuItem(text = { Text("Download policy") }, onClick = { open = false; onPolicy(show) })
            DropdownMenuItem(text = { Text("Refresh now") }, onClick = { open = false; onRefresh(show) })
            DropdownMenuItem(text = { Text("Unsubscribe") }, onClick = { open = false; onRemove(show) })
        }
    }
}


private fun androidx.compose.foundation.lazy.LazyListScope.podcastPaging(state: PodcastState, onMore: () -> Unit, onRetry: () -> Unit) {
    if (state.error != null) item(key = "podcast-error") { PodcastPagingError(state.error, onRetry) }
    else if (state.nextOffset != null) item(key = "podcast-more") { LaunchedEffect(state.nextOffset) { onMore() }; PodcastPagingSpinner() }
}

private fun androidx.compose.foundation.lazy.grid.LazyGridScope.podcastPaging(state: PodcastState, onMore: () -> Unit, onRetry: () -> Unit) {
    if (state.error != null) item(key = "podcast-error") { PodcastPagingError(state.error, onRetry) }
    else if (state.nextOffset != null) item(key = "podcast-more") { LaunchedEffect(state.nextOffset) { onMore() }; PodcastPagingSpinner() }
}

@Composable internal fun PodcastStatus(content: @Composable ColumnScope.() -> Unit) = Column(Modifier.fillMaxSize().padding(32.dp), verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically), horizontalAlignment = Alignment.CenterHorizontally, content = content)
@Composable internal fun PodcastPagingSpinner() = AccessibleLoadingIndicator("Loading more podcast shows", Modifier.fillMaxWidth().height(72.dp), size = 26.dp)
@Composable internal fun PodcastPagingError(message: String, retry: () -> Unit) = Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) { Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant); Button(onClick = retry) { Text("Retry") } }
private fun episodeCount(show: PodcastShow) = "${show.episodeCount} ${if (show.episodeCount == 1) "episode" else "episodes"}"
