package net.kdhmedia.mochil.local

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class LocalMediaIndexTest {
    @Test fun indexRoundTripsExtractedMetadataAndTimestamps() {
        val item = LocalMediaItem("content://tree", "content://document", "fallback.mp3", "mp3", "audio/mpeg", 42, LocalMediaKind.Audio, 123, "Real title", "Artist", "Album", 61.5)
        assertEquals(listOf(item), decodeLocalMediaIndex(encodeLocalMediaIndex(listOf(item))))
    }

    @Test fun malformedAndOversizedIndexesAreDiscarded() {
        assertTrue(decodeLocalMediaIndex("not-json".encodeToByteArray()).isEmpty())
        assertTrue(decodeLocalMediaIndex(ByteArray(MAX_INDEX_BYTES + 1)).isEmpty())
    }
}
