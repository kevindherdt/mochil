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
    suspend fun find(publication: Publication, query: String, maximumResults: Int = 200): List<Locator> {
        require(maximumResults in 1..500)
        val iterator = publication.search(query.trim()) ?: return emptyList()
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
