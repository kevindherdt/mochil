package net.kdhmedia.mochil.catalogue

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class BookSummaryParserTest {
    @Test fun `catalogue progress is retained for ordinary publication cards`() {
        val book = JSONObject("""{"id":"one","library_id":"books","title":"One","reading_progress":{"fraction":0.48,"finished":false}}""").toBookSummary()
        assertEquals(0.48, requireNotNull(book.readingFraction), 0.0001)
        assertFalse(book.finished)
    }

    @Test fun `continue reading response uses its top level progress`() {
        val book = JSONObject("""{"id":"one","library_id":"books","title":"One","fraction":0.72,"finished":true}""").toBookSummary()
        assertEquals(0.72, requireNotNull(book.readingFraction), 0.0001)
        assertTrue(book.finished)
    }

    @Test
    fun shelvedBooksCarryTheirStatusAndMapToTheMenuChoice() {
        val held = JSONObject("""{"id":"one","library_id":"books","title":"One","reading_progress":{"fraction":0.3,"status":"on_hold"}}""").toBookSummary()
        assertEquals("on_hold", held.status)
        assertEquals(ReadingStatus.OnHold, held.readingStatus())
        assertEquals(ReadingStatus.Read, held.withReadingStatus(ReadingStatus.Read).readingStatus())
        assertEquals(ReadingStatus.Reading, held.withReadingStatus(ReadingStatus.Reading).readingStatus())
        assertEquals(ReadingStatus.Unread, held.withReadingStatus(ReadingStatus.Unread).readingStatus())
        assertEquals(ReadingStatus.Dropped, held.withReadingStatus(ReadingStatus.Dropped).readingStatus())
    }

    @Test
    fun theHomeListReadsTheStatusFromTheTopLevelOfAReadingEntry() {
        val held = JSONObject("""{"id":"one","library_id":"books","title":"One","fraction":0.2,"status":"dropped"}""").toBookSummary()
        assertEquals(ReadingStatus.Dropped, held.readingStatus())
    }

    @Test
    fun aStreamableComicCarriesItsPageStreamUrlAndPageCount() {
        val book = JSONObject("""{"id":"one","library_id":"manga","title":"One","format":"cbz","page_count":42,"page_stream_url":"/stream/manga/one?page={pageNumber}"}""").toBookSummary()
        assertEquals("/stream/manga/one?page={pageNumber}", book.pageStreamURL)
        assertEquals(42, book.pageCount)
    }

    @Test
    fun aNonStreamableComicHasNoPageStreamUrl() {
        val book = JSONObject("""{"id":"one","library_id":"manga","title":"One","format":"cbz"}""").toBookSummary()
        assertEquals(null, book.pageStreamURL)
        assertEquals(0, book.pageCount)
    }
}
