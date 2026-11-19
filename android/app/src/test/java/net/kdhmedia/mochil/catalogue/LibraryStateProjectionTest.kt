package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Test

class LibraryStateProjectionTest {
    private val visible = LibrarySummary("books", "Books", "books", 12)
    private val hidden = LibrarySummary("video", "Video", "movies", 3)

    @Test fun `shell projection ignores catalogue churn`() {
        val before = LibraryState(libraries = listOf(visible), selected = visible)
        val after = before.copy(
            books = listOf(BookSummary("one", "books", "One", emptyList(), "", null, "epub", "application/epub+zip", "", "", 1)),
            loadingMore = true,
            query = "one",
        )

        assertEquals(before.shellProjection(), after.shellProjection())
    }

    @Test fun `visibility projection ignores selection and catalogue churn`() {
        val before = LibraryState(allLibraries = listOf(visible, hidden), libraries = listOf(visible), selected = visible)
        val after = before.copy(
            selected = null,
            series = listOf(SeriesSummary("books", "Series", emptyList(), 1, null)),
            groupLoading = true,
        )

        assertEquals(before.visibilityProjection(), after.visibilityProjection())
        assertEquals(setOf("books"), before.visibilityProjection().visibleLibraryIds)
    }
}
