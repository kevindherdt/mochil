package net.kdhmedia.mochil.ui

import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudDownload
import androidx.compose.material.icons.filled.DownloadDone
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.automirrored.filled.PlaylistAdd
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.CustomAccessibilityAction
import androidx.compose.ui.semantics.customActions
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion
import kotlin.math.roundToLong

@Composable
internal fun MusicTrackRow(
    track: AudioTrack,
    artwork: String?,
    authorizationHeader: String?,
    modifier: Modifier = Modifier,
    position: Int? = null,
    downloaded: Boolean,
    favorite: Boolean,
    downloading: Boolean,
    onPlay: () -> Unit,
    onToggleDownload: () -> Unit,
    onToggleFavorite: () -> Unit,
    onPlayNext: () -> Unit,
    onAddToQueue: () -> Unit,
    playlistNames: List<String>,
    onAddToPlaylist: (String) -> Unit,
) {
    var menuExpanded by remember(track.libraryId, track.id) { mutableStateOf(false) }
    Box {
        Row(
            modifier.fillMaxWidth().combinedClickable(
                onClickLabel = "Play ${track.title}",
                onLongClickLabel = "Track options",
                role = Role.Button,
                onClick = onPlay,
                onLongClick = { menuExpanded = true },
            ).semantics {
                customActions = listOf(CustomAccessibilityAction("Track options") {
                    menuExpanded = true
                    true
                })
            }.padding(start = 20.dp, end = 16.dp, top = 7.dp, bottom = 7.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            if (artwork != null) {
                AuthenticatedArtwork(artwork, authorizationHeader, track.title, Modifier.size(50.dp))
            } else {
                Text(
                    position?.toString().orEmpty(),
                    Modifier.width(28.dp),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                )
            }
            Column(Modifier.weight(1f)) {
                Text(track.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(
                    track.artist.ifBlank { "Unknown artist" },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            if (track.durationSeconds > 0) {
                Text(formatTrackDuration(track.durationSeconds), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (downloading) if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(20.dp), strokeWidth = 2.dp) else CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
        }
        MusicTrackActionMenu(track, downloaded, favorite, downloading, menuExpanded, { menuExpanded = false }, onToggleDownload, onToggleFavorite, onPlayNext, onAddToQueue, playlistNames, onAddToPlaylist)
    }
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
internal fun MusicTrackActionMenu(
    track: AudioTrack,
    downloaded: Boolean,
    favorite: Boolean,
    downloading: Boolean,
    expanded: Boolean,
    onDismissRequest: () -> Unit,
    onToggleDownload: () -> Unit,
    onToggleFavorite: () -> Unit,
    onPlayNext: () -> Unit,
    onAddToQueue: () -> Unit,
    playlistNames: List<String>,
    onAddToPlaylist: (String) -> Unit,
) {
    var playlistPicker by remember { mutableStateOf(false) }
    var showInfo by remember { mutableStateOf(false) }
    if (expanded) ModalBottomSheet(onDismissRequest = onDismissRequest) {
        Text(track.title, Modifier.padding(horizontal = 24.dp, vertical = 8.dp), style = MaterialTheme.typography.titleLarge, maxLines = 2, overflow = TextOverflow.Ellipsis)
        MusicTrackSheetAction("Play next") { onDismissRequest(); onPlayNext() }
        MusicTrackSheetAction("Add to queue") { onDismissRequest(); onAddToQueue() }
        MusicTrackSheetAction("Add to playlist", { Icon(Icons.AutoMirrored.Filled.PlaylistAdd, contentDescription = null) }) { onDismissRequest(); playlistPicker = true }
        MusicTrackSheetAction(if (favorite) "Remove favorite" else "Favorite", { Icon(if (favorite) Icons.Default.Favorite else Icons.Default.FavoriteBorder, contentDescription = null) }) { onDismissRequest(); onToggleFavorite() }
        MusicTrackSheetAction(
            if (downloading) "Downloading…" else if (downloaded) "Remove download" else "Download",
            { Icon(if (downloaded) Icons.Default.DownloadDone else Icons.Default.CloudDownload, contentDescription = null) },
            enabled = !downloading,
        ) { onDismissRequest(); onToggleDownload() }
        if (trackMetadataDetails(track).isNotEmpty()) MusicTrackSheetAction("Track info", { Icon(Icons.Default.Info, contentDescription = null) }) { onDismissRequest(); showInfo = true }
        Spacer(Modifier.navigationBarsPadding().height(8.dp))
    }
    if (playlistPicker) PlaylistPickerDialog(
        trackTitle = track.title,
        playlistNames = playlistNames,
        onDismiss = { playlistPicker = false },
        onAdd = { name -> playlistPicker = false; onAddToPlaylist(name) },
    )
    if (showInfo) MusicTrackInfoDialog(track, onDismiss = { showInfo = false })
}

@Composable
private fun MusicTrackSheetAction(label: String, icon: (@Composable () -> Unit)? = null, enabled: Boolean = true, onClick: () -> Unit) {
    ListItem(
        headlineContent = { Text(label) },
        leadingContent = icon,
        modifier = Modifier.fillMaxWidth().heightIn(min = 56.dp).clickable(enabled = enabled, role = Role.Button, onClick = onClick),
        colors = androidx.compose.material3.ListItemDefaults.colors(containerColor = Color.Transparent),
    )
}

internal fun formatTrackDuration(seconds: Double): String {
    val total = seconds.coerceAtLeast(0.0).roundToLong()
    val hours = total / 3600
    val minutes = total % 3600 / 60
    val remainder = total % 60
    return if (hours > 0) "$hours:${minutes.toString().padStart(2, '0')}:${remainder.toString().padStart(2, '0')}"
    else "$minutes:${remainder.toString().padStart(2, '0')}"
}
