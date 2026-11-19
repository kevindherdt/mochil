package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Test

class WikiRecentTest {
    private fun page(path: String, modified: String, format: String = "md") = WikiEntry("page", path, path, format = format, modified = modified)

    @Test
    fun recentPagesAreTheNewestMarkdownPagesFirst() {
        val entries = listOf(
            page("a.md", "2026-10-01T00:00:00Z"),
            page("Work/b.md", "2026-10-31T08:00:00Z"),
            page("pic.png", "2026-10-31T09:00:00Z", format = "png"),
            WikiEntry("folder", path = "Work"),
            page("c.md", ""),
        )
        assertEquals(listOf("Work/b.md", "a.md"), recentWikiPages(entries).map { it.path })
    }

    @Test
    fun theListIsCappedAtTheLimit() {
        val entries = (1..9).map { page("p$it.md", "2026-10-0${it}T00:00:00Z") }
        assertEquals(3, recentWikiPages(entries, 3).size)
    }
}
