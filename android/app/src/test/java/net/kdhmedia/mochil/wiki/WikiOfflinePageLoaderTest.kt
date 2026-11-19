package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.cancel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.launch
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

class WikiOfflinePageLoaderTest {
    private val entries = List(128) { WikiEntry("page", "$it", "$it.md", "Page", "md") }
    private val snapshot = WikiOfflineSnapshot("vault", entries, "0.md", emptyList(), 1)

    @Test fun `large pages stop fetching after the first non fitting batch`() = runBlocking {
        var requested = 0
        val updates = mutableListOf<Int>()
        val content = "x".repeat(1_000_000)
        val pages = loadBoundedWikiPages(snapshot, entries, 4, { batch ->
            requested += batch.size
            batch.map { WikiOfflinePage(it.path, WikiPage(it.id, it.title, content, "r", true), 1) }
        }, { count, _ -> updates += count })
        assertEquals(4, requested)
        assertEquals(listOf("0", "1"), pages.map { it.page.id })
        assertEquals(listOf(0, 2), updates)
    }

    @Test fun `small pages retain full target order and progress`() = runBlocking {
        val targets = entries.take(9)
        val updates = mutableListOf<Int>()
        val pages = loadBoundedWikiPages(snapshot, targets, 4, { batch ->
            batch.map { WikiOfflinePage(it.path, WikiPage(it.id, it.title, "text", "r", true), 1) }
        }, { count, total -> assertEquals(9, total); updates += count })
        assertEquals(targets.map { it.id }, pages.map { it.page.id })
        assertFalse(pages.any { it.page.writable })
        assertEquals(listOf(0, 4, 8, 9), updates)
    }

    @Test fun `oversized target list cannot fetch beyond the page limit`() = runBlocking {
        val targets = List(MAX_OFFLINE_PAGES + 10) { WikiEntry("page", "$it", "$it.md", "Page", "md") }
        var requested = 0
        val pages = loadBoundedWikiPages(snapshot, targets, 3, { batch ->
            requested += batch.size
            batch.map { WikiOfflinePage(it.path, WikiPage(it.id, it.title, "text", "r", true), 1) }
        }, { _, total -> assertEquals(targets.size, total) })
        assertEquals(MAX_OFFLINE_PAGES, requested)
        assertEquals(MAX_OFFLINE_PAGES, pages.size)
    }

    @Test fun `cancellation from a non cooperating batch cannot publish progress or return a snapshot`() = runBlocking {
        val updates = mutableListOf<Int>()
        var returned = false
        var calls = 0
        val job = launch {
            loadBoundedWikiPages(snapshot, entries, 4, { batch ->
                calls++
                currentCoroutineContext().cancel()
                batch.map { WikiOfflinePage(it.path, WikiPage(it.id, it.title, "text", "r", true), 1) }
            }, { count, _ -> updates += count })
            returned = true
        }
        job.join()
        assertEquals(1, calls)
        assertEquals(listOf(0), updates)
        assertFalse(returned)
    }

    @Test fun `already cancelled loader cannot fetch or publish initial progress`() = runBlocking {
        var calls = 0
        val updates = mutableListOf<Int>()
        val job = launch {
            currentCoroutineContext().cancel()
            loadBoundedWikiPages(snapshot, entries, 4, { calls++; emptyList() }, { count, _ -> updates += count })
        }
        job.join()
        assertEquals(0, calls)
        assertEquals(emptyList<Int>(), updates)
    }
}
