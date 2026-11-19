package net.kdhmedia.mochil.webdav

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class WebDavIdentityTest {
    /**
     * The cache file name used to be `href.hashCode()`, a 32-bit hash: two different books could land
     * on one file and the second opened the first one's content. These two happened to collide.
     */
    @Test
    fun `different resources never share an identity`() {
        assertNotEquals(
            webDavResourceId("https://dav.example/books/Aa.pdf", "same-etag"),
            webDavResourceId("https://dav.example/books/BB.pdf", "same-etag"),
        )
    }

    @Test
    fun `a replaced file gets a new identity`() {
        assertNotEquals(
            webDavResourceId("https://dav.example/books/one.pdf", "\"v1\""),
            webDavResourceId("https://dav.example/books/one.pdf", "\"v2\""),
        )
    }

    @Test
    fun `the same resource and version keep their identity`() {
        assertEquals(
            webDavResourceId("https://dav.example/books/one.pdf", "\"v1\""),
            webDavResourceId("https://dav.example/books/one.pdf", "\"v1\""),
        )
    }
}
