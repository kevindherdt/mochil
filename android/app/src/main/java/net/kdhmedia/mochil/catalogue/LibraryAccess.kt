package net.kdhmedia.mochil.catalogue

import kotlinx.coroutines.flow.StateFlow

/**
 * The download queue as a screen sees it. Deliberately not part of [LibraryState]: its progress ticks
 * arrive several times a second, so a banner collects this where it is drawn and a tick recomposes
 * nothing else on the shelf.
 */
val LibraryViewModel.downloadQueueItems: StateFlow<List<DownloadQueueItem>>
    get() = downloadQueue.queue

fun LibraryViewModel.queueBook(book: BookSummary) = downloadQueue.queueBook(book)
fun LibraryViewModel.removeQueued(book: BookSummary) = downloadQueue.removeQueued(book)
fun LibraryViewModel.clearFinishedDownloads() = downloadQueue.clearFinished()
