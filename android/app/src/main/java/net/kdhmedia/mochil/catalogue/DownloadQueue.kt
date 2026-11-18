package net.kdhmedia.mochil.catalogue

/** Where one queued book stands. A failed book stays in the list with its reason, so nothing vanishes silently. */
enum class DownloadQueueState { Waiting, Downloading, Done, Failed }

data class DownloadQueueItem(
    val book: BookSummary,
    val state: DownloadQueueState = DownloadQueueState.Waiting,
    val done: Long = 0,
    val total: Long = 0,
    val error: String? = null,
) {
    val key: String get() = "${book.libraryId}:${book.id}"
}

/** The book the queue is fetching right now, if any. */
fun activeDownload(items: List<DownloadQueueItem>): DownloadQueueItem? =
    items.firstOrNull { it.state == DownloadQueueState.Downloading }

/** The next book to fetch: the queue is sequential so only one book is ever in flight. */
fun nextWaitingDownload(items: List<DownloadQueueItem>): DownloadQueueItem? =
    items.firstOrNull { it.state == DownloadQueueState.Waiting }

/** A book that is queued but not finished yet, used to refuse adding the same book twice. */
fun isQueued(items: List<DownloadQueueItem>, book: BookSummary): Boolean {
    val key = "${book.libraryId}:${book.id}"
    return items.any { it.key == key && (it.state == DownloadQueueState.Waiting || it.state == DownloadQueueState.Downloading) }
}

/**
 * Replaces one item in place, chosen by its key. Position matters: the list is the reader's order,
 * and a book that finished must not jump to the front while the rest are still waiting.
 *
 * A book can appear twice — finished once, then queued again — so the update goes to the entry that
 * is still in flight when there is one, and only to the last entry otherwise. Keying on the first
 * match alone would start the *finished* row a second time and leave the new one waiting for ever.
 */
fun withDownloadUpdate(
    items: List<DownloadQueueItem>,
    key: String,
    transform: (DownloadQueueItem) -> DownloadQueueItem,
): List<DownloadQueueItem> {
    val unfinished = items.indexOfFirst {
        it.key == key && (it.state == DownloadQueueState.Waiting || it.state == DownloadQueueState.Downloading)
    }
    val index = if (unfinished >= 0) unfinished else items.indexOfLast { it.key == key }
    if (index < 0) return items
    return items.toMutableList().also { it[index] = transform(it[index]) }
}

/** Drops finished and failed entries, which is what "clear" means once the reader has seen them. */
fun withoutFinishedDownloads(items: List<DownloadQueueItem>): List<DownloadQueueItem> =
    items.filter { it.state == DownloadQueueState.Waiting || it.state == DownloadQueueState.Downloading }

/**
 * The line above the progress bar: which book, how far, and how many are left behind it. Null when
 * nothing is downloading, so the banner can disappear rather than show an idle queue.
 */
fun downloadQueueBannerText(items: List<DownloadQueueItem>): String? {
    val active = activeDownload(items) ?: return null
    val remaining = items.count { it.state == DownloadQueueState.Waiting }
    val position = items.indexOfFirst { it.key == active.key } + 1
    val title = active.book.title
    val progress = when {
        active.done <= 0 && active.total <= 0 -> "starting…"
        active.total > 0 -> "${bytes(active.done)} of ${bytes(active.total)}"
        else -> bytes(active.done)
    }
    val suffix = if (remaining > 0) " · $remaining more queued" else ""
    return "Downloading $title ($position of ${items.size}): $progress$suffix"
}

private fun bytes(count: Long): String = if (count < 1_048_576) "${count / 1024} KB" else "%.1f MB".format(count / 1_048_576.0)


/** The volume after this one, taken from whichever list the reader is looking at. */
fun nextBookInState(state: LibraryState, book: BookSummary): BookSummary? {
    val candidates = state.seriesBooks.takeIf { books -> books.any { it.id == book.id } } ?: state.books
    return nextBookInSeries(candidates, book)
}
