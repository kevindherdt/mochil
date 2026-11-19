package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.playback.PlaybackViewModel

/** Virtualized, remote-first live queue revealed only from TV Now Playing. */
@Composable
internal fun TvPlaybackQueue(
    playbackViewModel: PlaybackViewModel,
    artworkUrl: (AudioTrack) -> String,
    authorizationHeader: String?,
    onPersistQueue: (List<AudioTrack>) -> Unit,
) {
    // Stop observing queue mutations while Now Playing is not visible. This is
    // especially important on lower-powered TV hardware where the audio service
    // can continue updating independently of this screen.
    val queue by playbackViewModel.queueState.collectAsStateWithLifecycle()
    if (queue.tracks.isEmpty()) {
        Text("The playback queue is empty.", color = MaterialTheme.colorScheme.onSurfaceVariant)
        return
    }
    val initialIndex = tvQueueInitialIndex(queue.currentIndex, queue.tracks.size)
    val initialItemKey = queue.tracks[initialIndex].identityKey
    val positions = remember {
        TvGridPositionStore().apply { focus(TV_QUEUE_ROUTE, initialItemKey) }
    }
    val listState = rememberLazyListState(initialFirstVisibleItemIndex = initialIndex)
    LazyColumn(
        Modifier.fillMaxSize(),
        state = listState,
        contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 4.dp),
    ) {
        itemsIndexed(queue.tracks, key = { _, track -> track.identityKey }) { index, track ->
            var menu by remember(track.identityKey) { mutableStateOf(false) }
            Box(Modifier.padding(vertical = 4.dp)) {
                TvWideMediaCard(
                    title = track.title,
                    subtitle = if (index == queue.currentIndex) "Playing · ${track.artist}" else track.artist,
                    artwork = artworkUrl(track),
                    authorizationHeader = authorizationHeader,
                    onClick = { playbackViewModel.playQueueItem(track) },
                    active = index == queue.currentIndex,
                    onLongClick = { menu = true },
                    focusIdentity = TvGridFocusIdentity(TV_QUEUE_ROUTE, track.identityKey, positions),
                )
                TvContextDropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                    DropdownMenuItem(
                        text = { Text("Move earlier") },
                        enabled = index > 0,
                        onClick = { menu = false; onPersistQueue(playbackViewModel.moveQueueItem(track, -1)) },
                        modifier = Modifier.tvFocusBorder(),
                    )
                    DropdownMenuItem(
                        text = { Text("Move later") },
                        enabled = index < queue.tracks.lastIndex,
                        onClick = { menu = false; onPersistQueue(playbackViewModel.moveQueueItem(track, 1)) },
                        modifier = Modifier.tvFocusBorder(),
                    )
                    DropdownMenuItem(
                        text = { Text("Remove from queue") },
                        enabled = queue.tracks.size > 1,
                        onClick = {
                            menu = false
                            val before = playbackViewModel.queueState.value.tracks
                            val remaining = playbackViewModel.removeQueueItem(track)
                            tvQueueFocusAfterRemoval(before, remaining, track.identityKey, AudioTrack::identityKey)?.let {
                                positions.focus(TV_QUEUE_ROUTE, it.identityKey)
                            }
                            onPersistQueue(remaining)
                        },
                        modifier = Modifier.tvFocusBorder(),
                    )
                }
            }
        }
    }
}

private const val TV_QUEUE_ROUTE = "now-playing-queue"

internal fun tvQueueInitialIndex(currentIndex: Int, size: Int): Int =
    if (size <= 0) 0 else currentIndex.coerceIn(0, size - 1)

/** Prefer the item that takes the removed row's place, or the previous item at the end. */
internal fun <T> tvQueueFocusAfterRemoval(remaining: List<T>, removedIndex: Int): T? =
    if (remaining.isEmpty()) null else remaining[removedIndex.coerceIn(0, remaining.lastIndex)]

/** A menu may outlive its original row index; resolve against the latest queue. */
internal fun <T> tvQueueFocusAfterRemoval(
    before: List<T>,
    remaining: List<T>,
    targetKey: String,
    keyOf: (T) -> String,
): T? = remaining.firstOrNull { keyOf(it) == targetKey }
    ?: tvQueueFocusAfterRemoval(remaining, before.indexOfFirst { keyOf(it) == targetKey })
