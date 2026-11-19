package net.kdhmedia.mochil.webdav

import androidx.media3.common.C
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class WebDavStreamReferenceTest {
    @Test fun encryptedReferencePayloadRoundTripsOpaqueIdentity() {
        val value = WebDavStreamReference(webDavStreamToken("credential", "https://cloud.example/private/song.flac"), "connection", "credential", "https://cloud.example/private/song.flac", 42)
        assertEquals(listOf(value), decodeWebDavStreamReferences(encodeWebDavStreamReferences(listOf(value))))
        assertTrue(value.token.matches(Regex("^[a-f0-9]{36}$")))
        assertTrue(value.href !in value.token)
    }

    @Test fun malformedAndOversizedReferencePayloadsAreDiscarded() {
        assertTrue(decodeWebDavStreamReferences("not-json".encodeToByteArray()).isEmpty())
        assertTrue(decodeWebDavStreamReferences(ByteArray(MAX_STREAM_REFERENCE_BYTES + 1)).isEmpty())
    }

    @Test fun rangesAreExactAndOpenEndedOnlyWhenNeeded() {
        assertNull(buildWebDavRangeHeader(0, C.LENGTH_UNSET.toLong()))
        assertEquals("bytes=100-", buildWebDavRangeHeader(100, C.LENGTH_UNSET.toLong()))
        assertEquals("bytes=100-149", buildWebDavRangeHeader(100, 50))
    }
}
