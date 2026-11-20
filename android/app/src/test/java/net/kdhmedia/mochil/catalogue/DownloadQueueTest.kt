package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class DownloadQueueTest {
    private fun book(id: String, title: String = "Book $id", size: Long = 0) = BookSummary(
        id = id,
        libraryId = "books",
        title = title,
        authors = listOf("An Author"),
        series = "",
        seriesIndex = null,
        format = "epub",
        mediaType = "application/epub+zip",
        language = "en",
        description = "",
        size = size,
    )

    private fun item(id: String, state: DownloadQueueState = DownloadQueueState.Waiting, done: Long = 0, total: Long = 0) =
        DownloadQueueItem(book(id, size = total), state = state, done = done, total = total)

    @Test
    fun `only the first waiting book is offered, whatever finished before it`() {
        val queue = listOf(
            item("a1", DownloadQueueState.Done),
            item("a2", DownloadQueueState.Downloading),
            item("a3"),
            item("a4"),
        )
        assertEquals("a3", nextWaitingDownload(queue)?.book?.id)
        assertNull("a queue of finished books has nothing to fetch", nextWaitingDownload(queue.take(2)))
    }

    @Test
    fun `an update keeps the reader's order rather than moving a finished book to the front`() {
        val queue = listOf(item("a1"), item("a2"), item("a3"))
        val updated = withDownloadUpdate(queue, "books:a2") { it.copy(state = DownloadQueueState.Done) }
        assertEquals(listOf("a1", "a2", "a3"), updated.map { it.book.id })
        assertEquals(DownloadQueueState.Waiting, updated[0].state)
        assertEquals(DownloadQueueState.Done, updated[1].state)
    }

    @Test
    fun `clearing drops finished and failed books but keeps what is still coming`() {
        val queue = listOf(
            item("a1", DownloadQueueState.Done),
            item("a2", DownloadQueueState.Failed),
            item("a3", DownloadQueueState.Downloading),
            item("a4"),
        )
        assertEquals(listOf("a3", "a4"), withoutFinishedDownloads(queue).map { it.book.id })
    }

    @Test
    fun `the same book cannot be queued twice while it is still pending`() {
        val queue = listOf(item("a1", DownloadQueueState.Downloading))
        assertTrue(isQueued(queue, book("a1")))
        assertFalse("a finished book may be queued again after a delete", isQueued(listOf(item("a1", DownloadQueueState.Done)), book("a1")))
        assertFalse(isQueued(queue, book("a2")))
    }

    @Test
    fun `the banner names the book, its place in the queue and how far it has got`() {
        val queue = listOf(
            item("a1", DownloadQueueState.Downloading, done = 1_048_576, total = 3_145_728),
            item("a2"),
        )
        assertEquals("Downloading Book a1 (1 of 2): 1.0 MB of 3.0 MB · 1 more queued", downloadQueueBannerText(queue))
    }

    @Test
    fun `the banner says nothing when nothing is downloading, and still reads well before bytes arrive`() {
        assertNull(downloadQueueBannerText(listOf(item("a1"))))
        assertNull(downloadQueueBannerText(emptyList()))
        assertEquals("Downloading Book a1 (1 of 1): starting…", downloadQueueBannerText(listOf(item("a1", DownloadQueueState.Downloading))))
    }
}
