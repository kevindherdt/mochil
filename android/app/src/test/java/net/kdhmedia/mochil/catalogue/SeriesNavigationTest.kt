package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

private fun book(
    id: String,
    libraryId: String = "lib",
    series: String = "Series",
    seriesIndex: Double? = null,
    title: String = id,
) = BookSummary(
    id = id, libraryId = libraryId, title = title, authors = emptyList(), series = series,
    seriesIndex = seriesIndex, format = "epub", mediaType = "application/epub+zip", language = "en",
    description = "", size = 0L,
)

/** Characterizes LibraryViewModel.nextBookAfter before it is extracted into [nextBookInSeries]. */
class SeriesNavigationTest {
    @Test fun returnsNullWhenBookHasNoSeries() {
        val target = book("a", series = "")
        assertNull(nextBookInSeries(listOf(target), target))
    }

    @Test fun ordersBySeriesIndexThenTitleAndReturnsTheFollowingBook() {
        val a = book("a", seriesIndex = 1.0, title = "Alpha")
        val b = book("b", seriesIndex = 2.0, title = "Beta")
        val c = book("c", seriesIndex = 3.0, title = "Gamma")
        assertEquals(b, nextBookInSeries(listOf(c, a, b), a))
        assertEquals(c, nextBookInSeries(listOf(c, a, b), b))
    }

    @Test fun returnsNullForTheLastBookInSeries() {
        val a = book("a", seriesIndex = 1.0)
        val b = book("b", seriesIndex = 2.0)
        assertNull(nextBookInSeries(listOf(a, b), b))
    }

    @Test fun missingSeriesIndexSortsAfterNumberedEntries() {
        val a = book("a", seriesIndex = 1.0, title = "Alpha")
        val noIndex = book("no-index", seriesIndex = null, title = "Zeta")
        assertEquals(noIndex, nextBookInSeries(listOf(noIndex, a), a))
    }

    @Test fun ignoresBooksFromOtherLibrariesOrSeries() {
        val a = book("a", libraryId = "lib1", series = "Series", seriesIndex = 1.0)
        val otherLibrary = book("other-lib", libraryId = "lib2", series = "Series", seriesIndex = 2.0)
        val otherSeries = book("other-series", libraryId = "lib1", series = "Other", seriesIndex = 2.0)
        val match = book("match", libraryId = "lib1", series = "Series", seriesIndex = 2.0)
        assertEquals(match, nextBookInSeries(listOf(a, otherLibrary, otherSeries, match), a))
    }

    @Test fun whenTheBookItselfIsAbsentIndexOfFirstFallsThroughToTheFirstCandidate() {
        // Characterizes the original indexOfFirst(...)+1 arithmetic: a miss yields -1, so "+1"
        // lands on index 0 rather than producing no result. Preserved as-is during extraction.
        val a = book("a", seriesIndex = 1.0)
        val b = book("b", seriesIndex = 2.0)
        assertEquals(b, nextBookInSeries(listOf(b), a))
    }
}
