package net.kdhmedia.mochil.wiki

import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Test

class WikiOfflinePageBudgetTest {
    private val snapshot = WikiOfflineSnapshot("vault", emptyList(), "first.md", emptyList(), 1)
    private fun page(id: Int, content: String) = WikiOfflinePage(
        "$id.md", WikiPage("$id", "Title \"雪\"", content, "revision", true), 1,
    )

    @Test fun `incremental budget matches stored prefix including escaped and multibyte text`() {
        val pages = List(20) { page(it, "雪\n\"\\\t😀".repeat(20_000)) }
        val budget = WikiOfflinePageBudget(snapshot)
        val accepted = pages.takeWhile { budget.accept(it) }
        val stored = decodeWikiOfflineSnapshot(boundedWikiOfflinePayload(snapshot.copy(pages = pages))!!)!!
        assertEquals(stored.pages.map { it.page.id }, accepted.map { it.page.id })
        assertFalse(accepted.isEmpty())
        assertFalse(accepted.size == pages.size)
    }

    @Test fun `page count limit matches encoder`() {
        val pages = List(MAX_OFFLINE_PAGES + 1) { page(it, "small") }
        val budget = WikiOfflinePageBudget(snapshot)
        assertEquals(MAX_OFFLINE_PAGES, pages.takeWhile { budget.accept(it) }.size)
    }

    @Test fun `invalid pages and excessive tree are rejected`() {
        val budget = WikiOfflinePageBudget(snapshot)
        assertFalse(budget.accept(page(1, "text").copy(path = "")))
        assertFalse(budget.accept(page(1, "x".repeat(MAX_OFFLINE_PAGE_CHARS + 1))))
        assertThrows(IOException::class.java) { WikiOfflinePageBudget(snapshot.copy(libraryId = "")) }
        val entries = List(MAX_OFFLINE_ENTRIES + 1) { WikiEntry("page", "$it", "$it.md", "Page", "md") }
        assertThrows(IOException::class.java) { WikiOfflinePageBudget(snapshot.copy(entries = entries)) }
    }
}
