package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.BookSummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/**
 * The two lines a library card adds to its artwork. They follow the mock-up's card, minus its habit of
 * labelling everything "AI": a pill names the state or the format of that one book, and the progress line
 * only appears when there is progress to show.
 */
class BookCardTextTest {
    private fun book(
        title: String = "Neuromancer",
        series: String = "",
        seriesIndex: Double? = null,
        format: String = "epub",
        fraction: Double? = null,
        finished: Boolean = false,
        readAlong: Boolean = false,
        status: String = "",
    ) = BookSummary(
        id = "one", libraryId = "books", title = title, authors = listOf("William Gibson"),
        series = series, seriesIndex = seriesIndex, format = format, mediaType = "application/epub+zip", language = "en",
        description = "", size = 1, readingFraction = fraction, finished = finished, readAlong = readAlong, status = status,
    )

    /** The foot says where the reader is; a card with nothing else to add gets no pill at all. */
    @Test
    fun `an untouched book carries no pill`() {
        assertEquals("", bookCardTag(book()))
        assertNull(bookProgressLabel(book()))
    }

    @Test
    fun `a narrated book says so, not that some model is involved`() {
        assertEquals("Narrated", bookCardTag(book(readAlong = true, fraction = .3)))
    }

    /** Progress lives on the artwork; repeating it as "In progress" was the same fact twice. */
    @Test
    fun `progress is printed over the artwork and not repeated in a pill`() {
        val reading = book(fraction = .72)
        assertEquals("72% read", bookProgressLabel(reading))
        assertEquals("", bookCardTag(reading))
    }

    @Test
    fun `a finished book is complete on the artwork`() {
        assertEquals("Read", bookProgressLabel(book(fraction = .98, finished = true)))
        assertEquals("", bookCardTag(book(fraction = .98, finished = true)))
    }

    @Test
    fun `a parked book keeps its own word`() {
        assertEquals("On hold", bookCardTag(book(status = "on_hold")))
        assertEquals("Dropped", bookCardTag(book(status = "dropped")))
    }

    /** A file extension is not something a reader scans for; the pill stays in the reader's words. */
    @Test
    fun `a comic is still just something to read`() {
        assertEquals("", bookCardTag(book(format = "cbz")))
        assertEquals("", bookCardTag(book(format = "")))
    }

    @Test
    fun `a book in a series says where it sits`() {
        assertEquals("Book 3", bookSeriesLabel(book(series = "The Expanse", seriesIndex = 3.0)))
    }

    @Test
    fun `a stored index without a series is noise`() {
        // Every row carries a 1 by default, so 1984 must not claim to be Book 1 of nothing.
        assertEquals("", bookSeriesLabel(book(title = "1984", seriesIndex = 1.0)))
    }

    @Test
    fun `a series that only repeats the title is not a series`() {
        assertEquals("", bookSeriesLabel(book(title = "1984", series = "1984", seriesIndex = 1.0)))
    }

    @Test
    fun `a zero padded title prefix gives a position`() {
        assertEquals("Book 1", bookSeriesLabel(book(title = "01 Dies the Fire")))
        assertEquals("", bookSeriesLabel(book(title = "127 Hours")))
    }

    /** One page in is "Started", not "0% read": a number that rounds to nothing says nothing. */
    @Test
    fun `a book opened once says started and draws no bar`() {
        val barely = book(fraction = 0.004)
        assertEquals("Started", bookProgressLabel(barely))
        assertEquals(false, bookProgressBarShown(barely))
        val opened = book(fraction = 0.42)
        assertEquals("42% read", bookProgressLabel(opened))
        assertEquals(true, bookProgressBarShown(opened))
        assertEquals(true, bookProgressBarShown(book(finished = true)))
    }
}
