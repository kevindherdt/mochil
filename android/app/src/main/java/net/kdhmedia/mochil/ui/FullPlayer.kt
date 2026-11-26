package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.clickable
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.SkipNext
import androidx.compose.material.icons.filled.Redo
import androidx.compose.material.icons.filled.SkipPrevious
import androidx.compose.material.icons.filled.Undo
import androidx.compose.material.icons.automirrored.filled.QueueMusic
import androidx.compose.material.icons.filled.Repeat
import androidx.compose.material.icons.filled.RepeatOne
import androidx.compose.material.icons.filled.Shuffle
import androidx.compose.material.icons.filled.Timer
import androidx.compose.material.icons.filled.BookmarkAdd
import androidx.compose.material.icons.filled.DeleteOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.HorizontalDivider
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.StateFlow
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.ui.theme.MochilChrome
import net.kdhmedia.mochil.playback.PlaybackPlayerState
import net.kdhmedia.mochil.playback.PlaybackTimelineState
import net.kdhmedia.mochil.playback.chapterAt
import java.util.Locale

@Composable
fun FullPlayer(
    state: PlaybackPlayerState,
    timeline: StateFlow<PlaybackTimelineState>,
    artworkUrl: String,
    authorizationHeader: String?,
    onToggle: () -> Unit,
    onPrevious: () -> Unit,
    onNext: () -> Unit,
    onSeek: (Long) -> Unit,
    onOpenQueue: () -> Unit,
    onToggleRepeat: () -> Unit,
    onToggleShuffle: () -> Unit,
    onCycleSpeed: () -> Unit,
    onSetSleepTimer: (Int?) -> Unit,
    onAddAudiobookBookmark: () -> Unit,
    onRemoveAudiobookBookmark: (String) -> Unit,
) {
    val track = state.current ?: return
    var sleepMenuExpanded by remember { mutableStateOf(false) }
    val timelineState by timeline.collectAsStateWithLifecycle()
    Column(
        Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding()
            .padding(start = 28.dp, top = 18.dp, end = 28.dp, bottom = 28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Box(Modifier.fillMaxWidth(0.72f).aspectRatio(1f)) {
            AuthenticatedArtwork(artworkUrl, authorizationHeader, track.title, Modifier.fillMaxWidth().aspectRatio(1f))
        }
        Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(track.title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.SemiBold, textAlign = TextAlign.Center, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Text(track.artist.ifBlank { "Unknown artist" }, style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            if (track.album.isNotBlank()) Text(track.album, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1, overflow = TextOverflow.Ellipsis)
            playerSourceFacts(track)?.let { Text(it, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
        }
        FullPlayerTimeline(timeline, track.identityKey, onSeek)
        // Sheet 2's transport: 15 seconds either way, then previous / the big accent play / next.
        val durationMs = timelineState.durationMs.coerceAtLeast(0L)
        val skipBack = { onSeek((timelineState.positionMs - 15_000).coerceAtLeast(0L)) }
        val skipForward = { onSeek((timelineState.positionMs + 15_000).coerceAtMost(durationMs)) }
        Row(horizontalArrangement = Arrangement.spacedBy(14.dp), verticalAlignment = Alignment.CenterVertically) {
            SkipSecondsButton(Icons.Default.Undo, "Back 15 seconds", skipBack)
            IconButton(onClick = onPrevious, modifier = Modifier.size(52.dp)) { Icon(Icons.Default.SkipPrevious, contentDescription = "Previous track") }
            // One filled circle, the screen's only solid button: the mock-up's accent play.
            Box(
                Modifier.size(72.dp).clip(androidx.compose.foundation.shape.CircleShape)
                    .background(MaterialTheme.colorScheme.primary)
                    .clickable(onClickLabel = if (state.isPlaying) "Pause" else "Play", role = Role.Button, onClick = onToggle),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    if (state.isPlaying) Icons.Default.Pause else Icons.Default.PlayArrow,
                    contentDescription = if (state.isPlaying) "Pause" else "Play",
                    modifier = Modifier.size(36.dp), tint = MaterialTheme.colorScheme.onPrimary,
                )
            }
            IconButton(onClick = onNext, modifier = Modifier.size(52.dp)) { Icon(Icons.Default.SkipNext, contentDescription = "Next track") }
            SkipSecondsButton(Icons.Default.Redo, "Forward 15 seconds", skipForward)
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            if (showsShuffle(track.libraryKind)) {
                IconButton(onClick = onToggleShuffle) {
                    Icon(Icons.Default.Shuffle, contentDescription = if (state.shuffleEnabled) "Disable shuffle" else "Enable shuffle", tint = if (state.shuffleEnabled) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            if (track.libraryKind != "podcasts") {
                TextButton(onClick = onOpenQueue) {
                    Icon(Icons.AutoMirrored.Filled.QueueMusic, contentDescription = null)
                    Text("Queue", Modifier.padding(start = 8.dp))
                }
            }
            IconButton(onClick = onToggleRepeat) {
                Icon(
                    if (state.repeatMode == androidx.media3.common.Player.REPEAT_MODE_ONE) Icons.Default.RepeatOne else Icons.Default.Repeat,
                    contentDescription = when (state.repeatMode) {
                        androidx.media3.common.Player.REPEAT_MODE_ALL -> "Repeat all enabled"
                        androidx.media3.common.Player.REPEAT_MODE_ONE -> "Repeat one enabled"
                        else -> "Enable repeat"
                    },
                    tint = if (state.repeatMode == androidx.media3.common.Player.REPEAT_MODE_OFF) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.primary,
                )
            }
        }
        if (track.libraryKind == "audiobooks") {
            AudiobookNavigation(state, timeline, onSeek, onAddAudiobookBookmark, onRemoveAudiobookBookmark)
        }
        if (showsPlaybackSpeed(track.libraryKind)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                TextButton(onClick = onCycleSpeed) { Text("${playbackSpeedLabel(state.speed)}× speed") }
                Box {
                    TextButton(onClick = { sleepMenuExpanded = true }) {
                        Icon(Icons.Default.Timer, contentDescription = null)
                        Text(
                            if (state.sleepRemainingMs > 0) "${sleepMinutes(state.sleepRemainingMs)} min" else "Sleep timer",
                            Modifier.padding(start = 8.dp),
                        )
                    }
                    DropdownMenu(expanded = sleepMenuExpanded, onDismissRequest = { sleepMenuExpanded = false }) {
                        if (state.sleepRemainingMs > 0) DropdownMenuItem(text = { Text("Turn off") }, onClick = { sleepMenuExpanded = false; onSetSleepTimer(null) })
                        listOf(15, 30, 45, 60, 90).forEach { minutes ->
                            DropdownMenuItem(text = { Text("$minutes minutes") }, onClick = { sleepMenuExpanded = false; onSetSleepTimer(minutes) })
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun FullPlayerTimeline(timeline: StateFlow<PlaybackTimelineState>, itemKey: String, onSeek: (Long) -> Unit) {
    val state by timeline.collectAsStateWithLifecycle()
    var draggedPosition by remember(itemKey) { mutableFloatStateOf(-1f) }
    val duration = state.durationMs.coerceAtLeast(1L)
    val displayed = if (draggedPosition >= 0f) draggedPosition.toLong() else state.positionMs
    Column {
        Slider(
            value = displayed.coerceIn(0, duration).toFloat(),
            onValueChange = { draggedPosition = it },
            onValueChangeFinished = {
                if (draggedPosition >= 0f) onSeek(draggedPosition.toLong())
                draggedPosition = -1f
            },
            valueRange = 0f..duration.toFloat(),
        )
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(playbackClock(displayed), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(playbackClock(state.durationMs), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

/** The mock-up's 15-second skip: the loop arrow with the number in it, a small target beside the transport. */
@Composable
internal fun SkipSecondsButton(icon: androidx.compose.ui.graphics.vector.ImageVector, description: String, onClick: () -> Unit) {
    Box(
        Modifier.size(MochilChrome.touchMin).clip(androidx.compose.foundation.shape.CircleShape)
            .clickable(onClickLabel = description, role = Role.Button, onClick = onClick)
            .semantics { contentDescription = description },
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, contentDescription = null, modifier = Modifier.size(MochilChrome.touchMin / 2), tint = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = .55f))
        Text("15", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface)
    }
}

internal fun playerSourceFacts(track: net.kdhmedia.mochil.audio.AudioTrack): String? {
    if (track.playbackMode == "converted") {
        val bitrate = when (track.playbackQuality) { "low" -> 80; "medium" -> 128; else -> 192 }
        return "AAC · $bitrate kbps · Converted"
    }
    return buildList {
    track.container.trim().takeIf(String::isNotBlank)?.let { add(it.uppercase(Locale.ROOT)) }
    if (track.bitDepth > 0) add("${track.bitDepth}-bit")
    if (track.sampleRate >= 1_000) add(formatSampleRate(track.sampleRate))
    }.takeIf { it.isNotEmpty() }?.joinToString(" · ")
}

@Composable
private fun AudiobookNavigation(
    state: PlaybackPlayerState,
    timeline: StateFlow<PlaybackTimelineState>,
    onSeek: (Long) -> Unit,
    onAddBookmark: () -> Unit,
    onRemoveBookmark: (String) -> Unit,
) {
    val track = state.current ?: return
    val position by timeline.collectAsStateWithLifecycle()
    val currentChapter = chapterAt(track.chapters, position.positionMs)
    var expanded by remember(track.id) { mutableStateOf(false) }
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        Box {
            TextButton(onClick = { expanded = true }) {
                Text(currentChapter?.title ?: "Chapters", maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
            DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                track.chapters.forEachIndexed { index, chapter ->
                    DropdownMenuItem(
                        text = {
                            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                                Text(chapter.title.ifBlank { "Chapter ${index + 1}" }, Modifier.weight(1f), maxLines = 1, overflow = TextOverflow.Ellipsis)
                                Text(
                                    playbackClock((chapter.startSeconds.coerceAtLeast(0.0) * 1_000).toLong()),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        },
                        onClick = { expanded = false; onSeek((chapter.startSeconds.coerceAtLeast(0.0) * 1_000).toLong()) },
                    )
                }
                if (track.chapters.isNotEmpty() && state.audiobookBookmarks.isNotEmpty()) HorizontalDivider()
                state.audiobookBookmarks.forEach { bookmark ->
                    DropdownMenuItem(
                        text = {
                            Column {
                                Text(bookmark.label.ifBlank { "Bookmark" }, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                Text(playbackClock(bookmark.positionMs), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        },
                        onClick = { expanded = false; onSeek(bookmark.positionMs) },
                        trailingIcon = {
                            IconButton(onClick = { onRemoveBookmark(bookmark.id) }) {
                                Icon(Icons.Default.DeleteOutline, contentDescription = "Remove bookmark at ${playbackClock(bookmark.positionMs)}")
                            }
                        },
                    )
                }
            }
        }
        IconButton(onClick = onAddBookmark) {
            Icon(Icons.Default.BookmarkAdd, contentDescription = "Bookmark current audiobook position")
        }
    }
}

internal fun playbackSpeedLabel(speed: Float): String = if (speed % 1f == 0f) speed.toInt().toString() else "%.2f".format(Locale.ROOT, speed).trimEnd('0')
private fun sleepMinutes(milliseconds: Long): Long = ((milliseconds.coerceAtLeast(0) + 59_999L) / 60_000L).coerceAtLeast(1L)

internal fun showsShuffle(libraryKind: String): Boolean = libraryKind == "music"
internal fun showsPlaybackSpeed(libraryKind: String): Boolean = libraryKind == "audiobooks" || libraryKind == "podcasts"
