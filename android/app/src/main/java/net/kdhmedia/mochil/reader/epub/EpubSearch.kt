package net.kdhmedia.mochil.reader.epub

import org.readium.r2.shared.ExperimentalReadiumApi
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication
import org.readium.r2.shared.publication.services.search.search
import org.readium.r2.shared.util.getOrElse

data class EpubSearchState(
    val searching: Boolean = false,
    val query: String = "",
    val results: List<Locator> = emptyList(),
    val error: String? = null,
)

@OptIn(ExperimentalReadiumApi::class)
object EpubSearch {
    /**
     * Searches the book for a phrase.
     *
     * Readium matches the query literally, so a remembered phrase with the wrong punctuation finds
     * nothing: the book reads `He want to kill me, Julie?` and searching `want to kill me?` returns
     * zero results, in a book that plainly contains those words. Rather than teach the reader to
     * guess the punctuation of a line they half remember, a literal miss is retried once with
     * punctuation treated as a word break -- only on a miss, so a real match is never displaced.
     */
    suspend fun find(publication: Publication, query: String, maximumResults: Int = 200): List<Locator> {
        require(maximumResults in 1..500)
        val literal = query.trim()
        if (literal.isEmpty()) return emptyList()
        val exact = scan(publication, literal, maximumResults)
        if (exact.isNotEmpty()) return exact
        val relaxed = relaxQuery(literal)
        if (relaxed.isEmpty() || relaxed == literal) return emptyList()
        return scan(publication, relaxed, maximumResults)
    }

    private suspend fun scan(publication: Publication, query: String, maximumResults: Int): List<Locator> {
        val iterator = publication.search(query) ?: return emptyList()
        return try {
            buildList {
                while (size < maximumResults) {
                    val page = iterator.next().getOrElse { error(it.message) } ?: break
                    addAll(page.locators.take(maximumResults - size))
                }
            }
        } finally {
            iterator.close()
        }
    }
}

/**
 * The same phrase with punctuation treated as a word break, for the retry above. Apostrophes inside
 * a word are kept, so `don't` does not become `don t`; everything else that is not a letter or a
 * digit becomes a space, and runs of spaces collapse.
 */
internal fun relaxQuery(query: String): String =
    query
        .replace(Regex("[^\\p{L}\\p{N}']+"), " ")
        .replace(Regex("(?<!\\p{L})'|'(?!\\p{L})"), " ")
        .replace(Regex("\\s+"), " ")
        .trim()
