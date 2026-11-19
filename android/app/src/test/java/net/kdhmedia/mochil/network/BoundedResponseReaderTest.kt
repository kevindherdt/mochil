package net.kdhmedia.mochil.network

import java.io.ByteArrayInputStream
import java.io.IOException
import java.io.InputStream
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class BoundedResponseReaderTest {
    @Test fun `reader accepts exact ceiling and preserves unicode`() {
        val content = "Mochil · 本"
        val bytes = content.encodeToByteArray()
        assertEquals(content, ByteArrayInputStream(bytes).readBoundedResponseText(bytes.size))
    }

    @Test fun `reader rejects payload beyond ceiling`() {
        assertThrows(IOException::class.java) {
            ByteArrayInputStream(ByteArray(8193)).readBoundedResponseText(8192)
        }
    }

    @Test fun `oversized stream consumes only the permitted bytes and one overflow probe`() {
        for (maximum in listOf(1, 7, 8191, 8192, 8193, 65536)) {
            var consumed = 0
            val stream = object : InputStream() {
                override fun read(): Int { consumed++; return 65 }
                override fun read(bytes: ByteArray, offset: Int, length: Int): Int {
                    bytes.fill(65, offset, offset + length)
                    consumed += length
                    return length
                }
            }
            assertThrows(IOException::class.java) { stream.readBoundedResponseText(maximum) }
            assertEquals(maximum + 1, consumed)
        }
    }

    @Test fun `short reads preserve UTF8 split across byte boundaries`() {
        val text = "本📚é\r\n".repeat(2000)
        val source = ByteArrayInputStream(text.encodeToByteArray())
        val stream = object : InputStream() {
            override fun read() = source.read()
            override fun read(bytes: ByteArray, offset: Int, length: Int) = source.read(bytes, offset, minOf(1, length))
        }
        assertEquals(text, stream.readBoundedResponseText(text.encodeToByteArray().size))
    }

    @Test fun `large configured ceiling does not overflow arithmetic for short responses`() {
        assertEquals("ok", ByteArrayInputStream("ok".encodeToByteArray()).readBoundedResponseText(Int.MAX_VALUE))
        assertEquals("", ByteArrayInputStream(byteArrayOf()).readBoundedResponseText(1))
        assertThrows(IllegalArgumentException::class.java) { ByteArrayInputStream(byteArrayOf()).readBoundedResponseText(0) }
    }
}
