package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.flow.StateFlow
import net.kdhmedia.mochil.catalogue.DownloadQueueItem
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.activeDownload
import net.kdhmedia.mochil.catalogue.downloadQueueItems
import net.kdhmedia.mochil.catalogue.downloadQueueBannerText

/**
 * Says that a book is being fetched to open, with how far it has got, so a slow open never looks like nothing happened.
 *
 * The queue is collected HERE, around the text and the bar, not by the screen that hosts the banner:
 * its progress ticks arrive several times a second and a tick must not recompose the shelf behind it.
 * The banner's own height still comes from this composable, so a banner appearing or disappearing lays
 * the list out again as it always did.
 */
@Composable
fun OpeningBanner(state: LibraryState, queue: StateFlow<List<DownloadQueueItem>>, modifier: Modifier = Modifier) {
    val queued by queue.collectAsStateWithLifecycle()
    // Opening a book wins the banner: the reader is waiting on that one. The queue shows otherwise,
    // so a background download is never silent.
    val queueText = if (state.openingBookId == null) downloadQueueBannerText(queued) else null
    if (state.openingBookId == null && queueText == null) return
    Surface(modifier.fillMaxWidth(), color = MaterialTheme.colorScheme.secondaryContainer, tonalElevation = 3.dp) {
        Column(Modifier.padding(horizontal = 20.dp, vertical = 10.dp)) {
            Text(
                queueText ?: openingText(state),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSecondaryContainer,
            )
            val active = activeDownload(queued)
            when {
                state.openingTotal > 0 ->
                    LinearProgressIndicator(progress = { (state.openingDone.toFloat() / state.openingTotal).coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth().padding(top = 6.dp))
                state.openingBookId != null -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(top = 6.dp))
                active != null && active.total > 0 ->
                    LinearProgressIndicator(progress = { (active.done.toFloat() / active.total).coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth().padding(top = 6.dp))
                else -> LinearProgressIndicator(Modifier.fillMaxWidth().padding(top = 6.dp))
            }
        }
    }
}

internal fun openingText(state: LibraryState): String {
    val title = state.openingTitle.ifBlank { "your book" }
    return when {
        state.openingDone <= 0 -> "Opening $title…"
        state.openingTotal > 0 -> "Opening $title: ${megabytes(state.openingDone)} of ${megabytes(state.openingTotal)}"
        else -> "Opening $title: ${megabytes(state.openingDone)}"
    }
}

private fun megabytes(bytes: Long): String = if (bytes < 1_048_576) "${bytes / 1024} KB" else "%.1f MB".format(bytes / 1_048_576.0)

/** The banner on screens that do not already read the library state, so they are not redrawn as the download moves. */
@Composable
fun OpeningBannerHost(libraryViewModel: LibraryViewModel, modifier: Modifier = Modifier) {
    val state by libraryViewModel.state.collectAsStateWithLifecycle()
    OpeningBanner(state, libraryViewModel.downloadQueueItems, modifier)
}
