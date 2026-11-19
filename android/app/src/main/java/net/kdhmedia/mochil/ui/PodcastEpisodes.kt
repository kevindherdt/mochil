package net.kdhmedia.mochil.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CloudDownload
import androidx.compose.material.icons.filled.DownloadDone
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.platform.LocalContext
import android.text.format.DateUtils
import android.text.format.Formatter
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.podcast.PodcastPlayedFilter
import net.kdhmedia.mochil.podcast.PodcastState
import net.kdhmedia.mochil.podcast.PodcastEpisodeSort
import net.kdhmedia.mochil.podcast.isCrediblePodcastDate
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
internal fun PodcastEpisodes(
    state: PodcastState,
    showArtwork: String,
    header: String?,
    onBack: () -> Unit,
    onFilter: (PodcastPlayedFilter) -> Unit,
    onSort: (PodcastEpisodeSort) -> Unit,
    onSetPlayed: (AudioTrack, Boolean) -> Unit,
    onPlay: (AudioTrack) -> Unit,
    onToggleDownload: (AudioTrack) -> Unit,
    onMore: () -> Unit,
    onRetry: () -> Unit,
) {
    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to podcast shows") }
            if (showArtwork.isNotBlank()) AuthenticatedArtwork(showArtwork, header, state.selectedShow?.title.orEmpty(), Modifier.size(48.dp).padding(end = 8.dp))
            Column(Modifier.weight(1f)) {
                Text(state.selectedShow?.title.orEmpty(), style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text("Episodes", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(state.episodeSort.label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            PodcastFilter(state.playedFilter, state.episodeSort, onFilter, onSort)
        }
        when {
            state.loading -> AccessibleLoadingIndicator("Loading episodes", Modifier.fillMaxSize())
            state.error != null && state.episodes.isEmpty() -> PodcastStatus { Text(state.error, color = MaterialTheme.colorScheme.onSurfaceVariant); Button(onClick = onRetry) { Text("Try again") } }
            state.episodes.isEmpty() -> PodcastStatus { Text("No ${state.playedFilter.name.lowercase()} episodes were found.", color = MaterialTheme.colorScheme.onSurfaceVariant) }
            else -> LazyColumn(Modifier.fillMaxSize()) {
                items(state.episodes, key = AudioTrack::id) { episode ->
                    PodcastEpisodeRow(
                        episode, episode.key() in state.downloadedIds,
                        state.downloadingId == episode.key(), state.resolvingPlaybackId == episode.key(), episode.key() in state.updatingPlayedIds,
                        { onPlay(episode) }, { onToggleDownload(episode) }, { onSetPlayed(episode, it) },
                    )
                }
                if (state.error != null) item(key = "episode-error") { PodcastPagingError(state.error, onRetry) }
                else if (state.nextCursor != null) item(key = "episode-more") { LaunchedEffect(state.nextCursor) { onMore() }; PodcastPagingSpinner() }
            }
        }
    }
}

@Composable
private fun PodcastFilter(selected: PodcastPlayedFilter, sort: PodcastEpisodeSort, onFilter: (PodcastPlayedFilter) -> Unit, onSort: (PodcastEpisodeSort) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    Box {
        IconButton(onClick = { expanded = true }) { Icon(Icons.Default.FilterList, contentDescription = "Sort and filter episodes", tint = if (selected == PodcastPlayedFilter.All && sort == PodcastEpisodeSort.Newest) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.primary) }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            Text("Order", modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            PodcastEpisodeSort.entries.forEach { option -> DropdownMenuItem(text = { Text(option.label) }, leadingIcon = { if (sort == option) Icon(Icons.Default.Check, contentDescription = null) }, onClick = { expanded = false; onSort(option) }) }
            HorizontalDivider()
            Text("Show", modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp), style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            PodcastPlayedFilter.entries.forEach { filter -> DropdownMenuItem(text = { Text(filter.name) }, leadingIcon = { if (selected == filter) Icon(Icons.Default.Check, contentDescription = null) }, onClick = { expanded = false; onFilter(filter) }) }
        }
    }
}

@Composable
private fun PodcastEpisodeRow(episode: AudioTrack, downloaded: Boolean, downloading: Boolean, resolvingPlayback: Boolean, updatingPlayed: Boolean, onPlay: () -> Unit, onDownload: () -> Unit, onSetPlayed: (Boolean) -> Unit) {
    val context = LocalContext.current
    Row(Modifier.fillMaxWidth().clickable(onClickLabel = "Play ${episode.title}", role = Role.Button, onClick = onPlay).padding(horizontal = 14.dp, vertical = 9.dp), horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(3.dp)) {
            Text(episode.title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Medium, maxLines = 2, overflow = TextOverflow.Ellipsis)
            if (episode.description.isNotBlank()) Text(episode.description, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 3, overflow = TextOverflow.Ellipsis)
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                if (episode.finished) Icon(Icons.Default.CheckCircle, contentDescription = "Played", modifier = Modifier.size(15.dp), tint = MaterialTheme.colorScheme.primary)
                Text(episodeProgress(episode), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            val details = buildList {
                if (isCrediblePodcastDate(episode.publishedAtMillis)) add(DateUtils.getRelativeTimeSpanString(episode.publishedAtMillis, System.currentTimeMillis(), DateUtils.MINUTE_IN_MILLIS).toString())
                if (episode.durationSeconds > 0) add(durationLabel(episode.durationSeconds))
                if (episode.sizeBytes > 0) add(Formatter.formatShortFileSize(context, episode.sizeBytes))
                if (episode.serverDownloaded) add("On server")
            }.joinToString(" · ")
            if (details.isNotBlank()) Text(details, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
        Row(horizontalArrangement = Arrangement.spacedBy(0.dp)) {
            if (resolvingPlayback) {
                val modifier = Modifier.size(20.dp).align(Alignment.CenterVertically).semantics {
                    contentDescription = "Opening ${episode.title} from synced position"
                }
                if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = modifier, strokeWidth = 2.dp)
                else CircularProgressIndicator(modifier, strokeWidth = 2.dp)
            }
            IconButton(onClick = { onSetPlayed(!episode.finished) }, enabled = !updatingPlayed) {
                if (updatingPlayed) {
                    if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                    else CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
                } else Icon(Icons.Default.CheckCircle, contentDescription = if (episode.finished) "Mark ${episode.title} as unplayed" else "Mark ${episode.title} as played", tint = if (episode.finished) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
            }
            IconButton(onClick = onDownload, enabled = !downloading) {
                if (downloading) {
                    if (LocalReduceMotion.current) CircularProgressIndicator(progress = { 1f }, modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                    else CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
                } else Icon(if (downloaded) Icons.Default.DownloadDone else Icons.Default.CloudDownload, contentDescription = if (downloaded) "Remove download for ${episode.title}" else "Download ${episode.title}", tint = if (downloaded) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

private fun episodeProgress(episode: AudioTrack): String = when {
    episode.finished -> "Played"
    episode.resumePositionSeconds > 0 && episode.durationSeconds > 0 -> "${((episode.resumePositionSeconds / episode.durationSeconds) * 100).toInt().coerceIn(1, 99)}% listened"
    else -> "Not played"
}

private fun durationLabel(seconds: Double): String {
    val total = seconds.toLong().coerceAtLeast(0)
    val hours = total / 3600
    val minutes = (total % 3600) / 60
    return if (hours > 0) "${hours}h ${minutes}m" else "${minutes} min"
}

private fun AudioTrack.key() = identityKey
