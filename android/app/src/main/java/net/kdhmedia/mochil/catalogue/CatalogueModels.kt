package net.kdhmedia.mochil.catalogue

import org.json.JSONArray
import org.json.JSONObject
import net.kdhmedia.mochil.network.requireAtMost

data class LibrarySummary(
    val id: String,
    val name: String,
    val kind: String,
    val count: Int,
)

data class BookSummary(
    val id: String,
    val libraryId: String,
    val title: String,
    val authors: List<String>,
    val series: String,
    val seriesIndex: Double?,
    val format: String,
    val mediaType: String,
    val language: String,
    val description: String,
    val size: Long,
    val readingFraction: Double? = null,
    val finished: Boolean = false,
    /** An EPUB that carries its own recorded narration. */
    val readAlong: Boolean = false,
    /** "on_hold" or "dropped" when the reader shelved the book; empty otherwise. */
    val status: String = "",
    /** A `{pageNumber}` URL template for OPDS-PSE page streaming; null when the server cannot stream this book. */
    val pageStreamURL: String? = null,
    val pageCount: Int = 0,
) : java.io.Serializable

data class SeriesSummary(
    val libraryId: String,
    val name: String,
    val authors: List<String>,
    val count: Int,
    val coverPath: String?,
)

data class AuthorSummary(
    val libraryId: String,
    val name: String,
    val count: Int,
    val coverPath: String?,
)

data class BookPage(
    val items: List<BookSummary>,
    val nextCursor: String?,
)

data class SeriesPage(
    val items: List<SeriesSummary>,
    val nextCursor: String?,
)

data class AuthorPage(
    val items: List<AuthorSummary>,
    val nextCursor: String?,
)

enum class PublicationGrouping { Continue, Series, Authors, Collections, Recent, Books }
enum class PublicationLayout { Covers, List }

internal fun JSONObject.toBookSummary(): BookSummary {
    val authorValues = (optJSONArray("authors") ?: JSONArray()).requireAtMost(MAX_PUBLICATION_AUTHORS, "Publication authors")
    val progress = optJSONObject("reading_progress")
    val fraction = when {
        progress != null && progress.has("fraction") -> progress.optDouble("fraction")
        has("fraction") -> optDouble("fraction")
        else -> Double.NaN
    }.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0)
    return BookSummary(
        id = getString("id"), libraryId = getString("library_id"), title = optString("title", "Untitled"),
        authors = List(authorValues.length()) { authorValues.getString(it) }, series = optString("series"),
        seriesIndex = if (has("series_index") && !isNull("series_index")) optDouble("series_index") else null,
        format = optString("format"), mediaType = optString("media_type", "application/octet-stream"),
        language = optString("language"), description = optString("description"), size = optLong("size"),
        readingFraction = fraction, finished = progress?.optBoolean("finished") ?: optBoolean("finished"),
        readAlong = optBoolean("read_along"),
        status = (progress ?: this).optString("status"),
        pageStreamURL = optString("page_stream_url").takeIf { it.isNotBlank() },
        pageCount = optInt("page_count"),
    )
}

internal const val MAX_PUBLICATION_AUTHORS = 32
