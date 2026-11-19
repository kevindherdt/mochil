package net.kdhmedia.mochil.ui

import java.nio.file.Files
import java.nio.file.Paths
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.LibraryState
import net.kdhmedia.mochil.catalogue.LibrarySummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class PublicationVisibleBooksTest {
    private fun book(id: String, title: String, authors: List<String> = emptyList(), libraryId: String = "books") =
        BookSummary(id, libraryId, title, authors, "", null, "epub", "book", "en", "", 0)

    private val library = LibrarySummary("books", "Books", "books", 2)

    @Test fun `an online shelf keeps the list the ViewModel published`() {
        val state = LibraryState(books = listOf(book("a", "Alpha")), downloadedOnly = false)
        assertSame("an online shelf must not copy its books to filter them", state.books, visibleBooks(state))
    }

    @Test fun `the downloaded shelf is filtered by library, query and bucket`() {
        val downloaded = listOf(
            book("a", "Alpha", listOf("Ada")),
            book("b", "Beta", listOf("Bob")),
            book("c", "Alpha Two", libraryId = "other"),
        )
        val state = LibraryState(selected = library, downloadedBooks = downloaded, downloadedOnly = true)
        assertEquals(listOf("a", "b"), visibleBooks(state).map { it.id })
        assertEquals(listOf("b"), visibleBooks(state.copy(query = "bob")).map { it.id })
        assertEquals(listOf("b"), visibleBooks(state.copy(bucket = "B")).map { it.id })
        assertEquals(listOf("b"), visibleBooks(state.copy(query = "BET")).map { it.id })
        assertTrue(visibleBooks(state.copy(bucket = "#")).isEmpty())
    }

    /**
     * The empty-state test and the grid/list branch each called this themselves, so one frame filtered the shelf three
     * times. The screen now reads it once through the remembered value; this pins that the branch uses that value.
     */
    @Test fun `the browse screen filters the shelf once per composition`() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/ui/PublicationScreen.kt"))
        assertTrue("the browse screen should hoist the shelf through rememberVisibleBooks", source.contains("val books = rememberVisibleBooks(state)"))
        assertTrue("the grid must draw the hoisted list", source.contains("BookGrid(books,"))
        assertTrue("the list must draw the hoisted list", source.contains("BookList(books,"))
        assertEquals("no branch may filter the shelf again", 0, Regex("Book(Grid|List)\\(visibleBooks\\(").findAll(source).count())
    }
}
