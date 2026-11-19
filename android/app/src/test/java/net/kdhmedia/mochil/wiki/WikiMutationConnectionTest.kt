package net.kdhmedia.mochil.wiki

import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.OutputStream
import java.net.HttpURLConnection
import java.net.URL
import org.junit.Assert.*
import org.junit.Test

class WikiMutationConnectionTest {
    private class Connection(val fault: String = "", val cleanupFails: Boolean = false) : HttpURLConnection(URL("http://example.invalid")) {
        val failure = IOException("Injected request failure")
        val cleanup = IOException("Injected cleanup failure")
        val bytes = ByteArrayOutputStream()
        var disconnected = false
        var closed = false
        override fun connect() = Unit
        override fun usingProxy() = false
        override fun disconnect() {
            disconnected = true
            if (cleanupFails) throw cleanup
        }
        override fun setRequestMethod(value: String) {
            if (fault == "method") throw failure
            super.setRequestMethod(value)
        }
        override fun getOutputStream(): OutputStream {
            if (fault == "open") throw failure
            return object : OutputStream() {
                override fun write(value: Int) {
                    if (fault == "write") throw failure
                    bytes.write(value)
                }
                override fun close() {
                    closed = true
                    if (fault == "close") throw failure
                }
            }
        }
    }

    @Test fun `setup failures release connections and preserve original exceptions`() {
        for (fault in listOf("method", "open", "write", "close")) {
            val connection = Connection(fault)
            val failure = runCatching { prepareWikiMutationConnection(connection, "POST", "fixture-token", "text") }.exceptionOrNull()
            assertSame(fault, connection.failure, failure)
            assertTrue(fault, connection.disconnected)
            assertEquals(fault, fault == "write" || fault == "close", connection.closed)
        }
    }

    @Test fun `cleanup failure does not mask request failure`() {
        val connection = Connection("write", cleanupFails = true)
        val failure = runCatching { prepareWikiMutationConnection(connection, "PUT", "fixture-token", "text") }.exceptionOrNull()
        assertSame(connection.failure, failure)
        assertSame(connection.cleanup, failure!!.suppressed.single())
    }

    @Test fun `successful setup transfers connection ownership and writes UTF8 without redirects`() {
        val connection = Connection()
        assertSame(connection, prepareWikiMutationConnection(connection, "PUT", "fixture-token", "日本語 📚"))
        assertFalse(connection.disconnected)
        assertTrue(connection.closed)
        assertEquals("日本語 📚", connection.bytes.toByteArray().decodeToString())
        assertEquals("PUT", connection.requestMethod)
        assertEquals("Bearer fixture-token", connection.getRequestProperty("Authorization"))
        assertEquals("application/json", connection.getRequestProperty("Content-Type"))
        assertFalse(connection.instanceFollowRedirects)
        assertEquals(10_000, connection.connectTimeout)
        assertEquals(20_000, connection.readTimeout)
    }

    @Test fun `bodyless setup does not acquire an output stream`() {
        val connection = Connection("open")
        prepareWikiMutationConnection(connection, "DELETE", "fixture-token", null)
        assertFalse(connection.doOutput)
        assertFalse(connection.closed)
        assertFalse(connection.disconnected)
    }
}
