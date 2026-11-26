package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.BookSummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The marks a library card puts on its artwork and the two lines under it. The web's card (web/src/ui/BookCard.tsx):
 * a kind glyph and a "Vol. N" plate, a bar while reading, a tick when read — and no pill of text under the author.
 */
class BookCardTextTest {
    private fun book(
        title: String = "Neuromancer",
        series: String = "",
        seriesIndex: Double? = null,
        fraction: Double? = null,
        finished: Boolean = false,
    ) = BookSummary(
        id = "one", libraryId = "books", title = title, authors = listOf("William Gibson"),
        series = series, seriesIndex = seriesIndex, format = "epub", mediaType = "application/epub+zip", language = "en",
        description = "", size = 1, readingFraction = fraction, finished = finished,
    )

    @Test
    fun `a series position is the web's plate, and only a real one`() {
        assertEquals("Vol. 3", bookVolumePlate(book(series = "The Expanse", seriesIndex = 3.0)))
        // Every row carries a 1 by default; a book outside a series must not claim to be volume one.
        assertNull(bookVolumePlate(book(seriesIndex = 1.0)))
        assertNull(bookVolumePlate(book(series = "The Expanse", seriesIndex = 2.5)))
        // A number that is the title ("1984") is never a position.
        assertNull(bookVolumePlate(book(title = "1984", series = "1984", seriesIndex = 1.0)))
        // A zero-padded title prefix is a position, as the web's titles.ts still reads it.
        assertEquals("Vol. 1", bookVolumePlate(book(title = "01 Dies the Fire")))
        assertEquals("Vol. 12", bookVolumePlate(book(title = "012 - The Long War")))
        // Not zero-padded: a leading number is part of the title, not a position.
        assertNull(bookVolumePlate(book(title = "12 - The Long War")))
        assertNull(bookVolumePlate(book(title = "1984", seriesIndex = 1.0)))
    }

    @Test
    fun `the bar is the only reading state, so any progress shows it, and never on a finished book`() {
        assertFalse(bookProgressBarShown(book()))
        // A book just marked "Reading" sits at 0.1% (ReadingStatus.Reading); the old card said "Started" there.
        assertTrue(bookProgressBarShown(book(fraction = 0.001)))
        assertTrue(bookProgressBarShown(book(fraction = 0.42)))
        assertFalse("the tick replaces the bar", bookProgressBarShown(book(fraction = 1.0, finished = true)))
    }
}
