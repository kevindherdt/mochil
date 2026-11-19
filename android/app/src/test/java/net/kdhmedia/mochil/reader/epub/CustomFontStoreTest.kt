package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CustomFontStoreTest {
    @Test fun validatesSupportedFontHeaders() {
        assertTrue(isValidReaderFont(byteArrayOf(0, 1, 0, 0), "ttf"))
        assertTrue(isValidReaderFont("OTTO".encodeToByteArray(), "otf"))
        assertFalse(isValidReaderFont("wOFF".encodeToByteArray(), "ttf"))
        assertFalse(isValidReaderFont("OTTO".encodeToByteArray(), "woff"))
    }
}
