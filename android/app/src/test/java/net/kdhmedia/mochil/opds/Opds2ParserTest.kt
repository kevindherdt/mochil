package net.kdhmedia.mochil.opds

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test

class Opds2ParserTest {
    @Test fun parsesPublicationsNavigationPaginationSearchAndArtwork() {
        val feed = Opds2Parser.parse(FEED.trimIndent().encodeToByteArray())
        assertEquals("OPDS 2 Library", feed.title)
        assertEquals("next.json", feed.nextLink?.href)
        assertEquals("search{?query}", feed.searchLink?.href)
        val book = feed.entries.first()
        assertEquals("A Book", book.title)
        assertEquals(listOf("Writer"), book.authors)
        assertEquals("book.epub", book.acquisitionLink?.href)
        assertEquals("cover.jpg", book.artworkLink?.href)
        assertNotNull(feed.entries.last().navigationLink)
    }

    @Test fun expandsCommonOpds2SearchTemplates() {
        assertEquals("search?query=space%20opera", expandOpdsSearchTemplate("search{?query}", "space%20opera"))
        assertEquals("search?q=space%20opera", expandOpdsSearchTemplate("search{?q}", "space%20opera"))
    }

    private companion object {
        const val FEED = """
            {
              "metadata":{"title":"OPDS 2 Library","identifier":"urn:library"},
              "links":[
                {"rel":"next","href":"next.json","type":"application/opds+json"},
                {"rel":"search","href":"search{?query}","type":"application/opds+json"}
              ],
              "publications":[{
                "metadata":{"title":"A Book","identifier":"urn:book","author":[{"name":"Writer"}]},
                "links":[{"rel":"http://opds-spec.org/acquisition","href":"book.epub","type":"application/epub+zip"}],
                "images":[{"href":"cover.jpg","type":"image/jpeg"}]
              }],
              "navigation":[{"title":"Fiction","href":"fiction.json","type":"application/opds+json"}]
            }
        """
    }
}
