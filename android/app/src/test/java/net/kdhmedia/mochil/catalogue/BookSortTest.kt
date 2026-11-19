package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Test

class BookSortTest {
    @Test
    fun theSortOnlyAppliesToTheUngroupedList() {
        assertEquals("author", LibraryState(grouping = PublicationGrouping.Books, sort = "author").bookSort())
        assertEquals("", LibraryState(grouping = PublicationGrouping.Books, sort = "").bookSort())
        assertEquals("", LibraryState(grouping = PublicationGrouping.Series, sort = "recent").bookSort())
        assertEquals("recent", LibraryState(grouping = PublicationGrouping.Recent).bookSort())
    }
}
