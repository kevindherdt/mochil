package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicState
import net.kdhmedia.mochil.audio.hasSameIdentity
import net.kdhmedia.mochil.audio.stableKey

internal fun MusicState.usesInlineTrackGroup() = layout == MusicLayout.List && selectedArtist == null &&
    grouping in setOf(MusicGrouping.Albums, MusicGrouping.Series, MusicGrouping.Authors)

@Composable
internal fun MusicExpandableGroupList(
    state: MusicState,
    artworkUrl: (AudioGroup) -> String,
    authorizationHeader: String?,
    onOpen: (AudioGroup) -> Unit,
    onClose: () -> Unit,
    onPlay: (AudioTrack) -> Unit,
    onLoadMore: () -> Unit,
    onRetry: () -> Unit,
    onToggleDownload: (AudioTrack) -> Unit,
    onToggleFavorite: (AudioTrack) -> Unit,
    onPlayNext: (AudioTrack) -> Unit,
    onAddToQueue: (AudioTrack) -> Unit,
    playlists: MusicPlaylistBindings,
) {
    val playlistNames = playlists.state.playlists.map { it.name }
    LazyColumn(Modifier.fillMaxSize()) {
        state.groups.forEach { group ->
            val groupKey = group.stableKey()
            val expanded = group.hasSameIdentity(state.selectedGroup)
            item(key = "group:$groupKey") {
                Row(
                    Modifier.fillMaxWidth().clickable(onClickLabel = if (expanded) "Collapse ${group.name}" else "Expand ${group.name}", role = Role.Button) {
                        if (expanded) onClose() else onOpen(group)
                    }.padding(horizontal = 20.dp, vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    AuthenticatedArtwork(artworkUrl(group), authorizationHeader, group.name, Modifier.size(54.dp))
                    Column(Modifier.weight(1f)) {
                        Text(group.name, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        Text(groupCaptionForInlineList(group, state.libraryKind), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    Icon(if (expanded) Icons.Default.ExpandLess else Icons.Default.ExpandMore, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            if (expanded && state.loading) item(key = "group-loading:$groupKey") {
                Box(Modifier.fillMaxWidth().height(64.dp), contentAlignment = Alignment.Center) {
                    if (LocalReduceMotion.current) {
                        CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
                    } else {
                        CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 2.dp)
                    }
                }
            }
            if (expanded && state.error != null) item(key = "group-error:$groupKey") { PagingError(state.error, onRetry) }
            if (expanded && !state.loading && state.error == null) itemsIndexed(state.tracks, key = { _, track -> "track:${track.libraryId}:${track.id}" }) { index, track ->
                val key = track.identityKey
                MusicTrackRow(
                    track = track, artwork = null, authorizationHeader = null, position = index + 1,
                    downloaded = key in state.downloadedIds, favorite = key in state.favoriteIds,
                    downloading = state.downloadingId == key, onPlay = { onPlay(track) },
                    onToggleDownload = { onToggleDownload(track) }, onToggleFavorite = { onToggleFavorite(track) },
                    onPlayNext = { onPlayNext(track) }, onAddToQueue = { onAddToQueue(track) },
                    playlistNames = playlistNames, onAddToPlaylist = { playlists.add(track, it) },
                    modifier = Modifier.padding(start = 24.dp),
                )
            }
        }
        if (state.selectedGroup == null && state.error != null) item(key = "groups-error") { PagingError(state.error, onRetry) }
        else if (state.selectedGroup == null && state.nextOffset != null) item(key = "groups-next") {
            LaunchedEffect(state.nextOffset) { onLoadMore() }
            Box(Modifier.fillMaxWidth().height(72.dp), contentAlignment = Alignment.Center) {
                if (LocalReduceMotion.current) {
                    CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(26.dp))
                } else {
                    CircularProgressIndicator(Modifier.size(26.dp))
                }
            }
        }
    }
}

private fun groupCaptionForInlineList(group: AudioGroup, libraryKind: String): String {
    val noun = if (libraryKind == "audiobooks") if (group.count == 1) "book" else "books" else if (group.count == 1) "track" else "tracks"
    return listOf(group.subtitle.takeIf(String::isNotBlank), "${group.count} $noun").filterNotNull().joinToString(" · ")
}
