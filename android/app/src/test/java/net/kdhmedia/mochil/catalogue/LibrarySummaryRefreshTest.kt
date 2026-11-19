package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Test

class LibrarySummaryRefreshTest {
    @Test
    fun `renamed selected library updates without clearing open books`() {
        val old = LibrarySummary("tvshows", "tv-series", "tv", 35)
        val renamed = LibrarySummary("tvshows", "TV Series", "tv", 36)
        val books = listOf(BookSummary("one", "tvshows", "One", emptyList(), "", null, "", "", "", "", 0))
        val current = LibraryState(allLibraries = listOf(old), libraries = listOf(old), selected = old, books = books)

        val refreshed = current.withRefreshedLibraries(listOf(renamed), listOf(renamed))

        assertEquals("TV Series", refreshed.selected?.name)
        assertEquals(36, refreshed.selected?.count)
        assertSame(books, refreshed.books)
    }
}
