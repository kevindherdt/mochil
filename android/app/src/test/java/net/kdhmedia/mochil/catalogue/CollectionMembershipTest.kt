package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CollectionMembershipTest {
    private val book = BookSummary(
        id = "b1", libraryId = "books", title = "Neuromancer", series = "Sprawl",
        authors = emptyList(), seriesIndex = 1.0, format = "epub", mediaType = "application/epub+zip", language = "", description = "", size = 0,
    )

    private fun collection(vararg members: CollectionMember) = PublicationCollection("c", "to read", "books", members.size, members.toList())

    @Test
    fun aBookAddedWholeSeriesIsHeldThroughItsSeries() {
        val c = collection(CollectionMember("series", "books", series = "Sprawl"))
        assertTrue(c.holdsThroughSeries(book))
        assertFalse(c.contains(CollectionMember("book", "books", bookId = "b1")))
    }

    @Test
    fun anotherSeriesOrLibraryDoesNotCount() {
        assertFalse(collection(CollectionMember("series", "books", series = "Other")).holdsThroughSeries(book))
        assertFalse(collection(CollectionMember("series", "manga", series = "Sprawl")).holdsThroughSeries(book))
        assertFalse(collection().holdsThroughSeries(null))
    }

    @Test
    fun collectionsAreSharedLayoutWithSeriesAndAuthors() {
        assertEquals(true, PublicationGrouping.Collections.usesGroupLayout())
        assertEquals(false, PublicationGrouping.Books.usesGroupLayout())
    }
}
