package net.kdhmedia.mochil.catalogue

/**
 * The book that follows [book] within its own series/library, ordered by series index
 * (missing indices sort last) then title. Null if [book] has no series, isn't present in
 * [candidates], or is the last entry.
 */
internal fun nextBookInSeries(candidates: List<BookSummary>, book: BookSummary): BookSummary? {
    if (book.series.isBlank()) return null
    val ordered = candidates.filter { it.libraryId == book.libraryId && it.series == book.series }
        .sortedWith(compareBy<BookSummary> { it.seriesIndex ?: Double.MAX_VALUE }.thenBy { it.title.lowercase() })
    return ordered.getOrNull(ordered.indexOfFirst { it.id == book.id } + 1)
}
