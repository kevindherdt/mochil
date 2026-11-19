package net.kdhmedia.mochil.opds

import org.junit.Assert.assertEquals
import org.junit.Test

class OpdsAcquisitionTest {
    @Test fun derivesReaderFormatsFromMediaTypesAndSafePaths() {
        assertEquals("epub", formatFor("application/epub+zip", "https://books.example/file"))
        assertEquals("pdf", formatFor("application/pdf; charset=binary", "https://books.example/file"))
        assertEquals("cbz", formatFor("application/vnd.comicbook+zip", "https://books.example/file"))
        assertEquals("mobi", formatFor("application/octet-stream", "https://books.example/title.mobi"))
    }
}
