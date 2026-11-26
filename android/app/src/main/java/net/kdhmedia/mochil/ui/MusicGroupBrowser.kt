package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CloudDownload
import androidx.compose.material.icons.filled.DownloadDone
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Shuffle
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.*

/** Artist drill-down, group detail, and group collection rendering for music shelves. */
@Composable
internal fun ArtistAlbums(
    state: MusicState,
    artworkUrl: (AudioGroup) -> String,
    header: String?,
    onOpen: (AudioGroup) -> Unit,
    onBack: () -> Unit,
    onRetry: () -> Unit,
    onPlayArtist: (Boolean) -> Unit,
) {
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to artists") }
            Column(Modifier.weight(1f)) {
                Text(state.selectedArtist?.name.orEmpty(), style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text("${state.artistAlbums.size} ${if (state.artistAlbums.size == 1) "album" else "albums"}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            IconButton(onClick = { onPlayArtist(false) }) { Icon(Icons.Default.PlayArrow, contentDescription = "Play artist") }
            IconButton(onClick = { onPlayArtist(true) }) { Icon(Icons.Default.Shuffle, contentDescription = "Shuffle artist") }
        }
        when {
            state.loading -> AccessibleLoadingIndicator("Loading artist", Modifier.fillMaxSize())
            state.error != null -> CenteredMusicStatus { Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant); Button(onClick = onRetry) { Text("Try again") } }
            state.artistAlbums.isEmpty() -> CenteredMusicStatus { Text("No albums were found for this artist.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            state.layout == MusicLayout.Covers -> GroupCoverGrid(state.copy(groups = state.artistAlbums, nextOffset = null), artworkUrl, header, onOpen, {}, onRetry)
            else -> GroupList(state.copy(groups = state.artistAlbums, nextOffset = null), artworkUrl, header, onOpen, {}, onRetry)
        }
    }
}

@Composable
internal fun GroupCoverGrid(state: MusicState, artworkUrl: (AudioGroup) -> String, header: String?, onOpen: (AudioGroup) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    LazyVerticalGrid(columns = GridCells.Adaptive(136.dp), contentPadding = PaddingValues(20.dp), horizontalArrangement = Arrangement.spacedBy(14.dp), verticalArrangement = Arrangement.spacedBy(22.dp)) {
        items(state.groups, key = AudioGroup::stableKey) { group ->
            MusicAlbumCard(
                title = group.name, subtitle = groupCaption(group, state.libraryKind),
                art = { AuthenticatedArtwork(artworkUrl(group), header, group.name, Modifier.fillMaxWidth().aspectRatio(1f)) },
                onClick = { onOpen(group) },
            )
        }
        if (state.error != null) item(key = "group-error") { PagingError(state.error, onRetry) }
        else if (state.nextOffset != null) item(key = "group-next") { LaunchedEffect(state.nextOffset) { onMore() }; PagingSpinner() }
    }
}

@Composable
internal fun GroupList(state: MusicState, artworkUrl: (AudioGroup) -> String, header: String?, onOpen: (AudioGroup) -> Unit, onMore: () -> Unit, onRetry: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        items(state.groups, key = AudioGroup::stableKey) { group ->
            Row(Modifier.fillMaxWidth().clickable(onClickLabel = "Open ${group.name}", role = Role.Button) { onOpen(group) }.padding(horizontal = 20.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically) {
                AuthenticatedArtwork(artworkUrl(group), header, group.name, Modifier.size(54.dp))
                Column(Modifier.weight(1f)) {
                    Text(group.name, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    Text(groupCaption(group, state.libraryKind), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
        audioPaging(state, onMore, onRetry)
    }
}

@Composable
internal fun GroupTracks(state: MusicState, artworkUrl: (AudioGroup) -> String, header: String?, onPlay: (AudioTrack) -> Unit, onBack: () -> Unit, onRetry: () -> Unit, onDownloadedOnly: (Boolean) -> Unit, onFavoritesOnly: (Boolean) -> Unit, onShowAllTracks: () -> Unit, onToggleDownload: (AudioTrack) -> Unit, onToggleFavorite: (AudioTrack) -> Unit, onPlayNext: (AudioTrack) -> Unit, onAddToQueue: (AudioTrack) -> Unit, playlists: MusicPlaylistBindings, onDownloadGroup: () -> Unit, onQueueGroup: (List<AudioTrack>, Boolean) -> Unit, onPlayGroup: (List<AudioTrack>, Boolean) -> Unit) {
    val visibleTracks = rememberVisibleTracks(state)
    Column(Modifier.fillMaxSize()) {
        // Two rows: the title keeps a usable width on a 320-360dp phone, where the artwork plus
        // four 48dp controls used to leave it ~12dp and ellipsize the album name away.
        Column(Modifier.fillMaxWidth()) {
            Row(Modifier.fillMaxWidth().padding(start = 8.dp, end = 8.dp, top = 12.dp, bottom = 2.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to audio groups") }
                state.selectedGroup?.let { AuthenticatedArtwork(artworkUrl(it), header, it.name, Modifier.size(56.dp)) }
                Column(Modifier.weight(1f)) {
                    Text(state.selectedGroup?.name.orEmpty(), style = MaterialTheme.typography.titleLarge, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    state.selectedGroup?.let { Text(groupCaption(it, state.libraryKind), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                }
                GroupActionMenu(state, visibleTracks, onDownloadedOnly, onFavoritesOnly, onShowAllTracks, onDownloadGroup, onQueueGroup)
            }
            Row(Modifier.fillMaxWidth().padding(start = 56.dp, end = 12.dp, bottom = 10.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                IconButton(onClick = { onPlayGroup(visibleTracks, false) }, enabled = visibleTracks.isNotEmpty()) { Icon(Icons.Default.PlayArrow, contentDescription = "Play group") }
                IconButton(onClick = { onPlayGroup(visibleTracks, true) }, enabled = visibleTracks.size > 1) { Icon(Icons.Default.Shuffle, contentDescription = "Shuffle group") }
            }
        }
        state.downloadMessage?.let { Text(it, Modifier.padding(horizontal = 20.dp, vertical = 4.dp), style = MaterialTheme.typography.bodySmall, color = if (it.startsWith("Downloaded") || it.contains("already")) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error) }
        state.actionMessage?.let { MusicActionNotice(it) }
        when {
            state.loading -> AccessibleLoadingIndicator(if (state.libraryKind == "audiobooks") "Loading audiobook series" else "Loading album", Modifier.fillMaxSize())
            state.error != null -> CenteredMusicStatus { Text(state.error); Button(onClick = onRetry) { Text("Try again") } }
            visibleTracks.isEmpty() -> CenteredMusicStatus { Text(if (state.downloadedOnly) "No downloaded audio in this group." else "This audio group is empty.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            else -> TrackRows(state, visibleTracks, onPlay, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlists)
        }
    }
}

@Composable
private fun TrackRows(state: MusicState, tracks: List<AudioTrack>, onPlay: (AudioTrack) -> Unit, onToggleDownload: (AudioTrack) -> Unit, onToggleFavorite: (AudioTrack) -> Unit, onPlayNext: (AudioTrack) -> Unit, onAddToQueue: (AudioTrack) -> Unit, playlists: MusicPlaylistBindings) {
    val playlistNames = playlists.state.playlists.map { it.name }
    LazyColumn(Modifier.fillMaxSize()) {
        itemsIndexed(tracks, key = { _, track -> track.id }) { index, track ->
            MusicTrackRow(track, artwork = null, authorizationHeader = null, position = index + 1, downloaded = track.key() in state.downloadedIds, favorite = track.key() in state.favoriteIds, downloading = state.downloadingId == track.key(), onPlay = { onPlay(track) }, onToggleDownload = { onToggleDownload(track) }, onToggleFavorite = { onToggleFavorite(track) }, onPlayNext = { onPlayNext(track) }, onAddToQueue = { onAddToQueue(track) }, playlistNames = playlistNames, onAddToPlaylist = { playlists.add(track, it) })
        }
    }
}

internal fun androidx.compose.foundation.lazy.LazyListScope.audioPaging(state: MusicState, onMore: () -> Unit, onRetry: () -> Unit) {
    if (state.error != null) item(key = "music-error") { PagingError(state.error, onRetry) }
    else if (state.nextCursor != null || state.nextOffset != null) item(key = "music-next") { LaunchedEffect(state.nextCursor, state.nextOffset) { onMore() }; PagingSpinner() }
}

@Composable internal fun PagingSpinner() = AccessibleLoadingIndicator("Loading more audio", Modifier.fillMaxWidth().height(72.dp), size = 26.dp)

@Composable
internal fun PagingError(message: String, onRetry: () -> Unit) {
    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) { Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant); Button(onClick = onRetry) { Text("Retry") } }
}

@Composable
internal fun CenteredMusicStatus(content: @Composable ColumnScope.() -> Unit) {
    Column(Modifier.fillMaxSize().padding(32.dp), verticalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterVertically), horizontalAlignment = Alignment.CenterHorizontally, content = content)
}

@Composable
internal fun MusicActionNotice(message: String) {
    Text(message, Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
}

internal fun groupCaption(group: AudioGroup, libraryKind: String = "music") = listOf(
    group.subtitle.takeIf(String::isNotBlank),
    "${group.count} ${if (libraryKind == "audiobooks") { if (group.count == 1) "book" else "books" } else { if (group.count == 1) "track" else "tracks" }}",
).filterNotNull().joinToString(" · ")

private fun AudioTrack.key() = identityKey

@Composable
private fun GroupActionMenu(state: MusicState, tracks: List<AudioTrack>, onDownloadedOnly: (Boolean) -> Unit, onFavoritesOnly: (Boolean) -> Unit, onShowAll: () -> Unit, onDownloadGroup: () -> Unit, onQueueGroup: (List<AudioTrack>, Boolean) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val downloading = state.batchDownloadTotal > 0
    Row(verticalAlignment = Alignment.CenterVertically) {
        if (downloading) Text("${state.batchDownloadCompleted}/${state.batchDownloadTotal}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Box {
            IconButton(onClick = { expanded = true }, enabled = !downloading) {
                if (downloading) {
                    if (LocalReduceMotion.current) {
                        CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
                    } else {
                        CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 2.dp)
                    }
                } else {
                    Icon(Icons.Default.MoreVert, contentDescription = "Group options")
                }
            }
            DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                DropdownMenuItem(text = { Text("Play next") }, onClick = { expanded = false; onQueueGroup(tracks, true) })
                DropdownMenuItem(text = { Text("Add to queue") }, onClick = { expanded = false; onQueueGroup(tracks, false) })
                DropdownMenuItem(text = { Text(if (state.libraryKind == "audiobooks") "Download series" else "Download album") }, leadingIcon = { Icon(Icons.Default.CloudDownload, contentDescription = null) }, onClick = { expanded = false; onDownloadGroup() })
                HorizontalDivider()
                DropdownMenuItem(text = { Text("All tracks") }, onClick = { expanded = false; onShowAll() })
                DropdownMenuItem(text = { Text("Favorites") }, leadingIcon = { Icon(Icons.Default.Favorite, contentDescription = null) }, onClick = { expanded = false; onFavoritesOnly(true) })
                DropdownMenuItem(text = { Text("Downloaded") }, leadingIcon = { Icon(Icons.Default.DownloadDone, contentDescription = null) }, onClick = { expanded = false; onDownloadedOnly(true) })
            }
        }
    }
}

/**
 * An album or playlist: square artwork, the name, and one muted line — the web's MusicCard (web/src/ui/MusicCard.tsx). No
 * progress, no badge: a group has no position of its own, and the format of a whole album is not one fact. Equal text
 * heights come from two title lines and one subtitle line, not from a fixed height.
 */
@Composable
internal fun MusicAlbumCard(title: String, subtitle: String, art: @Composable () -> Unit, onClick: () -> Unit, modifier: Modifier = Modifier, onDismiss: (() -> Unit)? = null) {
    Column(
        modifier.gridCard().combinedClickable(
            onClickLabel = "Open $title", role = Role.Button, onClick = onClick,
            onLongClickLabel = if (onDismiss != null) "Remove $title from Discover" else null,
            onLongClick = onDismiss,
        ).padding(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Box(Modifier.fillMaxWidth().aspectRatio(1f).clip(RoundedCornerShape(8.dp))) { art() }
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, minLines = 2, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(subtitle.ifEmpty { " " }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, minLines = 1, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}
