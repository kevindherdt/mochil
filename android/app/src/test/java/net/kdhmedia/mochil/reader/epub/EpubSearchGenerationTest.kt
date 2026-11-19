package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * An EPUB search scans the whole publication, so it can finish seconds after the reader dismissed the sheet or typed a
 * different query. Only the newest search may publish; anything older (and anything dismissed) must be dropped.
 */
class EpubSearchGenerationTest {
    @Test
    fun theNewestSearchOwnsTheResult() {
        val generation = EpubSearchGeneration()
        val first = generation.start()
        assertTrue(generation.isCurrent(first))
        val second = generation.start()
        assertTrue(generation.isCurrent(second))
        assertFalse("the superseded search must not publish", generation.isCurrent(first))
    }

    @Test
    fun dismissingTheSheetAbandonsTheSearchInFlight() {
        val generation = EpubSearchGeneration()
        val running = generation.start()
        // Closing the search sheet takes the blank-query path, which claims the generation again.
        generation.start()
        assertFalse("a dismissed search must not reopen the sheet", generation.isCurrent(running))
    }

    @Test
    fun tokensAreNeverReissued() {
        val generation = EpubSearchGeneration()
        val tokens = List(3) { generation.start() }
        assertEquals("each claim must be distinct, or an old result could match a newer search", 3, tokens.distinct().size)
    }
}
