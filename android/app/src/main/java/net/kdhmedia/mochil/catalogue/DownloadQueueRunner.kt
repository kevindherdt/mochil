package net.kdhmedia.mochil.catalogue

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.updateAndGet
import kotlinx.coroutines.isActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Runs the download queue: one book at a time, in the order the reader added them.
 *
 * It owns the sequencing and the queue state, and publishes that state as [queue]. Progress arrives a
 * few times a second, so the queue is NOT copied into [LibraryState] — a screen that shows it collects
 * the flow where the queue is drawn, or every tick would recompose the whole shelf. Keeping it out of
 * the view model makes the part that can actually go wrong — a failed book stalling everything behind
 * it — testable without a server or a Compose host.
 */
class DownloadQueueRunner(
    private val scope: CoroutineScope,
    private val fetch: suspend (BookSummary, (done: Long, total: Long) -> Unit) -> Result<Unit>,
    /**
     * Called once when a book has finished fetching. That is a real change — the downloaded set grew —
     * and it is the only queue event allowed to cost the shelf a state write; progress ticks are not.
     */
    private val onFetched: () -> Unit = {},
) {
    private val items = MutableStateFlow<List<DownloadQueueItem>>(emptyList())
    val queue: StateFlow<List<DownloadQueueItem>> = items.asStateFlow()
    private var running = false

    /** Adds a book and starts it if nothing else is downloading. False when it is already queued. */
    fun enqueue(book: BookSummary): Boolean {
        if (isQueued(items.value, book)) return false
        update { it + DownloadQueueItem(book) }
        pump()
        return true
    }

    fun remove(book: BookSummary) {
        val key = "${book.libraryId}:${book.id}"
        update { list -> list.filterNot { it.key == key && it.state == DownloadQueueState.Waiting } }
    }

    fun clearFinished() = update { withoutFinishedDownloads(it) }

    /** Fetches the first waiting book and calls itself when that finishes, success or failure. */
    private fun pump() {
        if (running) return
        val next = nextWaitingDownload(items.value) ?: return
        running = true
        update { list ->
            withDownloadUpdate(list, next.key) {
                it.copy(state = DownloadQueueState.Downloading, done = 0, total = 0, error = null)
            }
        }
        scope.launch {
            var reported = 0L
            // A fetch that throws, or a scope cancelled mid-transfer, must still clear the flag —
            // otherwise one bad book wedges the queue for the rest of the session.
            try {
                fetch(next.book) { done, total ->
                    // A few updates a second are plenty for a progress line.
                    val now = System.currentTimeMillis()
                    if (now - reported >= 250) {
                        reported = now
                        // Progress only: refreshing the downloaded index here re-read and
                        // re-stat the whole shelf several times a second, which is exactly the
                        // work the queue's own state flow was separated out to avoid.
                        update { list -> withDownloadUpdate(list, next.key) { it.copy(done = done, total = total) } }
                    }
                }
                    .onSuccess {
                        update { list -> withDownloadUpdate(list, next.key) { item -> item.copy(state = DownloadQueueState.Done, done = item.total) } }
                        onFetched()
                    }
                    .onFailure { failure ->
                        // One refused book must not take the rest of the queue with it.
                        val reason = failure.message ?: "Could not download this book."
                        update { list -> withDownloadUpdate(list, next.key) { item -> item.copy(state = DownloadQueueState.Failed, error = reason) } }
                    }
            } catch (cancellation: CancellationException) {
                throw cancellation
            } catch (failure: Throwable) {
                update { list ->
                    withDownloadUpdate(list, next.key) { item ->
                        item.copy(state = DownloadQueueState.Failed, error = failure.message ?: "Could not download this book.")
                    }
                }
            } finally {
                running = false
                if (scope.isActive) pump()
            }
        }
    }

    // Progress arrives on the download thread while enqueue/remove run on the main thread, so the
    // read-modify-write has to be atomic: a progress tick must never resurrect an older list and
    // drop a book queued a moment ago.
    private fun update(transform: (List<DownloadQueueItem>) -> List<DownloadQueueItem>) {
        items.updateAndGet { transform(it) }
    }
}
