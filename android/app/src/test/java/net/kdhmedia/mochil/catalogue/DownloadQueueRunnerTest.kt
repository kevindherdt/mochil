package net.kdhmedia.mochil.catalogue

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DownloadQueueRunnerTest {
    private fun book(id: String) = BookSummary(
        id = id, libraryId = "books", title = "Book $id", authors = listOf("An Author"), series = "",
        seriesIndex = null, format = "epub", mediaType = "application/epub+zip", language = "en",
        description = "", size = 10,
    )

    @Test
    fun `one book at a time, and a refused book does not stop the queue`() = runBlocking {
        var active = 0
        var mostAtOnce = 0
        val release = CompletableDeferred<Unit>()
        val attempted = mutableListOf<String>()

        val runner = DownloadQueueRunner(
            scope = this,
            fetch = { item, onProgress ->
                attempted += item.id
                active++
                mostAtOnce = maxOf(mostAtOnce, active)
                onProgress(5, 10)
                // The first book is held open until the test lets it finish, so a second transfer
                // starting early would show up as two active downloads.
                if (item.id == "a") release.await()
                active--
                if (item.id == "b") Result.failure(IllegalStateException("the server refused it"))
                else Result.success(Unit)
            },
        )

        assertTrue(runner.enqueue(book("a")))
        assertTrue(runner.enqueue(book("b")))
        assertTrue(runner.enqueue(book("c")))
        delay(50)
        assertEquals("only the first book may be in flight", listOf("a"), attempted)

        release.complete(Unit)
        delay(200)

        assertEquals(listOf("a", "b", "c"), attempted)
        assertEquals("the queue must never run two transfers at once", 1, mostAtOnce)
        val states = runner.queue.value.map { it.book.id to it.state }
        assertEquals(
            listOf(
                "a" to DownloadQueueState.Done,
                "b" to DownloadQueueState.Failed,
                "c" to DownloadQueueState.Done,
            ),
            states,
        )
        assertTrue("a failed book keeps its reason", runner.queue.value[1].error!!.contains("refused"))
    }

    @Test
    fun `a fetch that throws does not wedge the queue`() = runBlocking {
        val attempted = mutableListOf<String>()
        val runner = DownloadQueueRunner(
            scope = this,
            fetch = { item, _ ->
                attempted += item.id
                if (item.id == "a") throw IllegalStateException("the reader blew up")
                Result.success(Unit)
            },
        )
        runner.enqueue(book("a"))
        runner.enqueue(book("b"))
        delay(200)
        assertEquals(listOf("a", "b"), attempted)
        assertEquals(DownloadQueueState.Failed, runner.queue.value[0].state)
        assertEquals(DownloadQueueState.Done, runner.queue.value[1].state)
    }

    @Test
    fun `a book queued again after finishing is a separate row`() = runBlocking {
        val runner = DownloadQueueRunner(
            scope = this,
            fetch = { _, _ -> delay(50); Result.success(Unit) },
        )
        runner.enqueue(book("a"))
        delay(300)
        val first = runner.queue.value
        assertEquals("the first transfer should have finished: $first", DownloadQueueState.Done, first.single().state)

        runner.enqueue(book("a"))
        delay(50)
        assertEquals("the finished row and the new one both stay", 2, runner.queue.value.size)
        delay(300)
        assertEquals(DownloadQueueState.Done, runner.queue.value.last().state)
    }

    @Test
    fun `the same book is not queued twice while it is still pending`() = runBlocking {
        val runner = DownloadQueueRunner(this, fetch = { _, _ -> Result.success(Unit) })
        assertTrue(runner.enqueue(book("a")))
        assertTrue("a duplicate is refused", !runner.enqueue(book("a")))
        assertEquals(1, runner.queue.value.size)
    }
}
