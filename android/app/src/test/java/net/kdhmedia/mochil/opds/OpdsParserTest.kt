package net.kdhmedia.mochil.opds

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.ByteArrayInputStream

class OpdsParserTest {
    @Test fun parsesAcquisitionArtworkPaginationAndSearch() {
        val feed = OpdsParser.parse(FEED.trimIndent().encodeToByteArray())
        assertEquals("Library", feed.title)
        assertEquals("next.xml", feed.nextLink?.href)
        assertEquals("search?q={searchTerms}", feed.searchLink?.href)
        assertEquals("A Book", feed.entries.single().title)
        assertEquals(listOf("Writer"), feed.entries.single().authors)
        assertEquals("book.epub", feed.entries.single().acquisitionLink?.href)
        assertEquals("cover.jpg", feed.entries.single().artworkLink?.href)
    }

    @Test fun ignoresEntriesWithoutAcquisitionWhenAdapted() {
        val entry = OpdsEntry("one", "Navigation", emptyList(), listOf(OpdsLink("subsection", "child.xml")))
        assertNull(entry.acquisitionLink)
    }

    @Test fun acceptsOnlyCredentialFreeHttpsAddresses() {
        assertEquals("https", validatedHttps("https://books.example/opds").scheme)
        assertTrue(runCatching { validatedHttps("http://books.example/opds") }.isFailure)
        assertTrue(runCatching { validatedHttps("https://user:secret@books.example/opds") }.isFailure)
    }

    @Test fun parsesOpenSearchAtomTemplate() {
        val xml = """<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/"><Url type="application/atom+xml;profile=opds-catalog" template="https://books.example/search?q={searchTerms}"/></OpenSearchDescription>"""
        assertEquals("https://books.example/search?q={searchTerms}", OpdsSearchParser.parseTemplate(xml.encodeToByteArray()))
    }

    @Test fun boundedReaderRejectsOversizedFeeds() {
        assertEquals(4, readBounded(ByteArrayInputStream(byteArrayOf(1, 2, 3, 4)), 4).size)
        assertTrue(runCatching { readBounded(ByteArrayInputStream(ByteArray(5)), 4) }.isFailure)
    }

    private companion object {
        const val FEED = """
            <?xml version="1.0"?>
            <feed xmlns="http://www.w3.org/2005/Atom">
              <id>urn:library</id><title>Library</title>
              <link rel="next" href="next.xml" />
              <link rel="search" type="application/atom+xml" href="search?q={searchTerms}" />
              <entry><id>urn:book</id><title>A Book</title><author><name>Writer</name></author>
                <link rel="http://opds-spec.org/image" href="cover.jpg" type="image/jpeg" />
                <link rel="http://opds-spec.org/acquisition" href="book.epub" type="application/epub+zip" />
              </entry>
            </feed>
        """
    }
}
