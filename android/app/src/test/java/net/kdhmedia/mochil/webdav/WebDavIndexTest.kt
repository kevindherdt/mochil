package net.kdhmedia.mochil.webdav

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WebDavIndexTest {
    @Test fun indexRoundTripsMediaWithoutTurningFilesIntoFolders() {
        val value = WebDavIndex("connection", "credential", listOf(WebDavEntry("https://cloud.example/a/Book.epub", "Book.epub", false, 42, "application/epub+zip", "tag", "today")), 123, true)
        assertEquals(listOf(value), decodeWebDavIndexes(encodeWebDavIndexes(listOf(value))))
        assertFalse(decodeWebDavIndexes(encodeWebDavIndexes(listOf(value))).single().entries.single().collection)
    }

    @Test fun malformedAndOversizedIndexesAreDiscarded() {
        assertTrue(decodeWebDavIndexes("not-json".encodeToByteArray()).isEmpty())
        assertTrue(decodeWebDavIndexes(ByteArray(MAX_INDEX_BYTES + 1)).isEmpty())
    }
}
