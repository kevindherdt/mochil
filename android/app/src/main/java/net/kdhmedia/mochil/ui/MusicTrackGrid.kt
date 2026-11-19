package net.kdhmedia.mochil.ui

import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicLayout
import net.kdhmedia.mochil.audio.MusicState

internal fun MusicState.usesTrackCoverGrid() = layout == MusicLayout.Covers &&
    (grouping == MusicGrouping.Favorites || grouping == MusicGrouping.RecentlyPlayed)

@Composable
internal fun MusicTrackCoverGrid(
    state: MusicState,
    tracks: List<AudioTrack>,
    artworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    onPlay: (AudioTrack) -> Unit,
    onToggleDownload: (AudioTrack) -> Unit,
    onToggleFavorite: (AudioTrack) -> Unit,
    onPlayNext: (AudioTrack) -> Unit,
    onAddToQueue: (AudioTrack) -> Unit,
    playlists: MusicPlaylistBindings,
) {
    val playlistNames = playlists.state.playlists.map { it.name }
    LazyVerticalGrid(
        columns = GridCells.Adaptive(142.dp),
        contentPadding = PaddingValues(20.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp),
        verticalArrangement = Arrangement.spacedBy(20.dp),
    ) {
        items(tracks, key = AudioTrack::identityKey) { track ->
            var menuExpanded by remember(track.libraryId, track.id) { mutableStateOf(false) }
            Box {
            Column(
                Modifier.combinedClickable(
                    onClickLabel = "Play ${track.title}",
                    onLongClickLabel = "Track options",
                    role = Role.Button,
                    onClick = { onPlay(track) },
                    onLongClick = { menuExpanded = true },
                ).semantics {
                    customActions = listOf(CustomAccessibilityAction("Track options") {
                        menuExpanded = true
                        true
                    })
                },
                verticalArrangement = Arrangement.spacedBy(7.dp),
            ) {
                AuthenticatedArtwork(
                    artworkUrl(track), authorizationHeader, track.title,
                    Modifier.fillMaxWidth().aspectRatio(1f),
                )
                Text(track.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(track.artist.ifBlank { "Unknown artist" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
                if (state.downloadingId == track.identityKey) {
                    androidx.compose.material3.LinearProgressIndicator(Modifier.fillMaxWidth().height(2.dp))
                }
            }
            MusicTrackActionMenu(
                track = track,
                downloaded = track.identityKey in state.downloadedIds,
                favorite = track.identityKey in state.favoriteIds,
                downloading = state.downloadingId == track.identityKey,
                expanded = menuExpanded,
                onDismissRequest = { menuExpanded = false },
                onToggleDownload = { onToggleDownload(track) },
                onToggleFavorite = { onToggleFavorite(track) },
                onPlayNext = { onPlayNext(track) },
                onAddToQueue = { onAddToQueue(track) },
                playlistNames = playlistNames,
                onAddToPlaylist = { playlists.add(track, it) },
            )
            }
        }
    }
}
