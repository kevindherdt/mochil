package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.BookSummary
import org.junit.Assert.assertEquals
import org.junit.Test

class DashboardReadingTitleTest {
    @Test fun mangaChapterIncludesItsSeriesOnHome() {
        assertEquals("Battle Royale — Chapter 0", dashboardReadingTitle(book("Chapter 0", "Battle Royale")))
    }

    @Test fun ordinaryPublicationTitleIsNotRepeated() {
        assertEquals("Volume 1", dashboardReadingTitle(book("Volume 1", "Battle Royale")))
    }

    private fun book(title: String, series: String) = BookSummary(
        id = "book", libraryId = "manga", title = title, series = series,
        authors = emptyList(), seriesIndex = null, format = "cbz",
        mediaType = "application/vnd.comicbook+zip", language = "", description = "", size = 0,
    )
}
