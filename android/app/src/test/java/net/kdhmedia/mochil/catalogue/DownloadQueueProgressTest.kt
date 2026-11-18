package net.kdhmedia.mochil.catalogue

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DownloadQueueProgressTest {
    private fun book(id: String) = BookSummary(
        id = id, libraryId = "books", title = "Book $id", authors = listOf("An Author"), series = "",
        seriesIndex = null, format = "epub", mediaType = "application/epub+zip", language = "en",
        description = "", size = 10,
    )

    /**
     * Progress ticks are not a change to the downloaded set. Calling back on every tick made the
     * shelf re-read and re-stat its whole offline index several times a second, which is the work the
     * queue's own state flow was separated out to avoid.
     */
    @Test
    fun `progress ticks never refresh the downloaded set`() = runBlocking {
        var refreshes = 0
        val finish = CompletableDeferred<Unit>()
        val runner = DownloadQueueRunner(
            scope = this,
            fetch = { _, onProgress ->
                // Enough ticks to cross several 250 ms progress batches, while the transfer stays
                // open so the assertion sees the middle of the download rather than its end.
                repeat(12) { step ->
                    onProgress(step.toLong(), 12)
                    delay(60)
                }
                finish.await()
                Result.success(Unit)
            },
            onFetched = { refreshes += 1 },
        )

        assertTrue(runner.enqueue(book("a")))
        delay(800)
        assertEquals("a progress tick must not refresh the shelf", 0, refreshes)

        finish.complete(Unit)
        delay(300)
        assertEquals("the completed download refreshes it exactly once", 1, refreshes)
    }
}
