package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** The reading-status chips of the Library header: All, In Progress, Unread (web/src/features/library/screens/LibraryScreen.tsx STATUS_CHIPS). */
class BookStatusFilterTest {
    @Test fun `the chips are named and sent as the web names and sends them`() {
        assertEquals(listOf("All", "In Progress", "Unread"), BookStatusFilter.entries.map { it.label })
        assertEquals(listOf("", "in_progress", "unread"), BookStatusFilter.entries.map { it.wire })
    }

    @Test fun `a value the app does not know is All`() {
        assertEquals(BookStatusFilter.All, BookStatusFilter.fromWire(""))
        assertEquals(BookStatusFilter.All, BookStatusFilter.fromWire("read"))
        assertEquals(BookStatusFilter.All, BookStatusFilter.fromWire("in_progress; drop"))
        assertEquals(BookStatusFilter.InProgress, BookStatusFilter.fromWire("in_progress"))
    }

    @Test fun `the books request carries status only when a filter is chosen`() {
        assertFalse(booksRequestQuery("books", null, null, "", "", "").contains("status"))
        assertFalse(booksRequestQuery("books", null, null, "", "", "", "").contains("status"))
        assertTrue(booksRequestQuery("books", null, null, "", "", "", "in_progress").contains("&status=in_progress"))
        assertTrue(booksRequestQuery("books", null, null, "", "", "", "unread").contains("&status=unread"))
    }

    @Test fun `a status the server would not understand is never sent`() {
        assertFalse(booksRequestQuery("books", null, null, "", "", "", "read&library=other").contains("status"))
        assertFalse(booksRequestQuery("books", null, null, "", "", "", "IN_PROGRESS").contains("status"))
    }

    @Test fun `status sits with the other filters and before the cursor`() {
        assertEquals(
            "?library=books&limit=60&q=dune&bucket=D&sort=recent&status=unread&cursor=c1",
            booksRequestQuery("books", null, "c1", "dune", "D", "recent", "unread"),
        )
    }

    @Test fun `the empty message names the filter and the noun`() {
        assertEquals("No books in progress.", BookStatusFilter.InProgress.emptyMessage("books"))
        assertEquals("No audiobooks in progress.", BookStatusFilter.InProgress.emptyMessage("audiobooks"))
        assertEquals("Every item here has been opened.", BookStatusFilter.Unread.emptyMessage("books"))
        assertEquals("This shelf has no books.", BookStatusFilter.All.emptyMessage("books"))
    }
}
