package net.kdhmedia.mochil.catalogue

import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Test

class NewDownloadDocumentTest {
    @Test fun `successful copy retains the created document`() {
        val written = mutableListOf<String>()
        val result = writeNewDownloadDocument({ "new" }, { written += it }, { throw AssertionError("Successful copy deleted") })
        assertEquals("new", result)
        assertEquals(listOf("new"), written)
    }

    @Test fun `failed copy discards only the new document and preserves the failure`() {
        val failure = IOException("Storage full")
        val deleted = mutableListOf<String>()
        val result = runCatching {
            writeNewDownloadDocument({ "new" }, { throw failure }, { deleted += it })
        }
        assertSame(failure, result.exceptionOrNull())
        assertEquals(listOf("new"), deleted)
    }

    @Test fun `creation failure never attempts cleanup`() {
        val failure = IOException("Folder unavailable")
        val result = runCatching {
            writeNewDownloadDocument<String>({ throw failure }, { throw AssertionError() }, { throw AssertionError() })
        }
        assertSame(failure, result.exceptionOrNull())
    }

    @Test fun `cleanup failure does not obscure the original copy error`() {
        val failure = IOException("Write failed")
        val cleanup = SecurityException("Permission revoked")
        val result = runCatching {
            writeNewDownloadDocument({ "new" }, { throw failure }, { throw cleanup })
        }
        assertSame(failure, result.exceptionOrNull())
        assertEquals(listOf(cleanup), failure.suppressed.toList())
    }
}
