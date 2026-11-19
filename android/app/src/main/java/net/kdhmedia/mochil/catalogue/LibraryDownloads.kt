package net.kdhmedia.mochil.catalogue

import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.withContext

/**
 * The download queue as the library sees it: add a book, remove one that has not started, clear what
 * finished. The sequencing itself lives in [DownloadQueueRunner]; this is only the wiring between
 * that runner, the view model's reader and the download store.
 *
 * Nothing here copies the queue into [LibraryState]. Its progress ticks arrive a few times a second,
 * and a screen that showed them from the state recomposed the whole shelf — controls, chips, the
 * filtered book list — for every tick. [queue] is the runner's own [StateFlow], so the banner can
 * collect it where it is drawn and a tick reaches nothing else. A finished download is a real
 * change, so that one still refreshes the downloaded set in the state.
 */
internal class LibraryDownloads(
    private val vm: LibraryViewModel,
    /**
     * Where a book comes from. Production goes through the opener so a queued book uses the same
     * download path as "read now"; a test can drive the queue without a server.
     */
    fetchOverride: (suspend (BookSummary, (done: Long, total: Long) -> Unit) -> Result<Unit>)? = null,
) {
    val runner = DownloadQueueRunner(
        scope = vm.viewModelScope,
        fetch = fetchOverride ?: { book, onProgress -> fetch(book, onProgress) },
        // A finished book is a real change to the shelf, so the downloaded set is refreshed once.
        onFetched = { vm.refreshDownloadedBooks() },
    )

    /** The live queue, in the reader's order; collect it where it is drawn, never through [LibraryState]. */
    val queue: StateFlow<List<DownloadQueueItem>> get() = runner.queue

    fun queueBook(book: BookSummary) {
        if (vm.credential == null) return vm.update { copy(openError = "This device is no longer connected.") }
        if (!runner.enqueue(book)) vm.update { copy(openError = "${book.title} is already in the queue.") }
    }

    fun removeQueued(book: BookSummary) = runner.remove(book)

    fun clearFinished() = runner.clearFinished()

    /**
     * The opener downloads and records the file; the Intent it returns is for "read now", which a
     * queued book does not need yet. Reusing it keeps one download path rather than two.
     */
    private suspend fun fetch(book: BookSummary, onProgress: (Long, Long) -> Unit): Result<Unit> {
        val saved = vm.credential ?: return Result.failure(IllegalStateException("This device is no longer connected."))
        return runCatching {
            withContext(Dispatchers.IO) {
                vm.opener.prepare(saved, book, nextBookInState(vm.mutableState.value, book), forceDownload = true, onProgress = onProgress)
            }
        }.map { }
    }
}
