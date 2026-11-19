package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EpubFontRuntimeTest {
    @Test
    fun `font family is stable for a stored font identity`() {
        assertEquals("MochilCustom_abc123", customFontFamily("abc123"))
    }

    @Test
    fun `truetype injection embeds the font without line wrapping`() {
        val script = customFontInjectionScript(
            CustomReaderFont("font-id", "Readable", "ttf"),
            byteArrayOf(0, 1, 2, 3),
        )

        assertTrue(script.contains("font-family:'MochilCustom_font-id'"))
        assertTrue(script.contains("data:font/ttf;base64,AAECAw=="))
        assertTrue(script.contains("format('truetype')"))
        assertFalse(script.contains('\n'))
    }

    @Test
    fun `opentype injection declares its actual browser format`() {
        val script = customFontInjectionScript(
            CustomReaderFont("serif", "Serif", "OTF"),
            byteArrayOf(79, 84, 84, 79),
        )

        assertTrue(script.contains("data:font/otf;base64,T1RUTw=="))
        assertTrue(script.contains("format('opentype')"))
    }
}
