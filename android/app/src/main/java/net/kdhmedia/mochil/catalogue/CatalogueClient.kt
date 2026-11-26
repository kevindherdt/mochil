package net.kdhmedia.mochil.catalogue

import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.network.MAX_ERROR_RESPONSE_BYTES
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import org.json.JSONArray

class CatalogueClient {
    fun libraries(credential: DeviceCredential): List<LibrarySummary> =
        PerformanceDiagnostics.measure("network", "load_libraries") { request(credential, "/api/android/v1/libraries") }.let { response ->
            val values = JSONArray(response.body).requireAtMost(MAX_LIBRARIES, "Library directory")
            List(values.length()) { index ->
                values.getJSONObject(index).run {
                    LibrarySummary(getString("id"), getString("name"), optString("kind"), optInt("count"))
                }
            }
        }

    fun books(credential: DeviceCredential, libraryId: String, cursor: String?, query: String = "", bucket: String = "", sort: String = "", status: String = ""): BookPage {
        return books(credential, libraryId, null, cursor, query, bucket, sort, status)
    }

    fun books(credential: DeviceCredential, libraryId: String, series: String?, cursor: String?, query: String = "", bucket: String = "", sort: String = "", status: String = ""): BookPage {
        val requestQuery = booksRequestQuery(libraryId, series, cursor, query, bucket, sort, status)
        return PerformanceDiagnostics.measure("network", "load_books") { request(credential, "/api/android/v1/books$requestQuery") }.let { response ->
            val values = JSONArray(response.body).requireAtMost(500, "Book page")
            BookPage(
                items = List(values.length()) { index -> values.getJSONObject(index).toBookSummary() },
                nextCursor = response.nextCursor,
            )
        }
    }

    fun continuing(credential: DeviceCredential, libraryId: String, query: String = ""): List<BookSummary> {
        val values = JSONArray(PerformanceDiagnostics.measure("network", "load_continue_reading") {
            request(credential, "/api/android/v1/reading?library=${encode(libraryId)}&limit=100")
        }.body).requireAtMost(100, "Continue reading")
        val books = List(values.length()) { index -> values.getJSONObject(index).toBookSummary() }
        val wanted = query.trim().lowercase()
        return if (wanted.isEmpty()) books else books.filter { book ->
            listOf(book.title, book.series, book.authors.joinToString()).any { wanted in it.lowercase() }
        }
    }

    fun authorBooks(credential: DeviceCredential, libraryId: String, author: String, cursor: String? = null): BookPage {
        val requestQuery = groupedBooksRequestQuery(libraryId, null, author, cursor, "", "", "", 500)
        return PerformanceDiagnostics.measure("network", "load_author_books") { request(credential, "/api/android/v1/books$requestQuery") }.asBookPage()
    }

    fun series(credential: DeviceCredential, libraryId: String, cursor: String?, query: String = "", bucket: String = ""): SeriesPage {
        val requestQuery = buildString {
            append("?library=").append(encode(libraryId)).append("&limit=60")
            if (query.isNotBlank()) append("&q=").append(encode(query.trim()))
            if (bucket.isNotBlank()) append("&bucket=").append(encode(bucket))
            if (!cursor.isNullOrBlank()) append("&cursor=").append(encode(cursor))
        }
        return PerformanceDiagnostics.measure("network", "load_series") { request(credential, "/api/android/v1/series$requestQuery") }.let { response ->
            val values = JSONArray(response.body).requireAtMost(60, "Series page")
            SeriesPage(
                items = List(values.length()) { index ->
                    values.getJSONObject(index).run {
                        val authorValues = (optJSONArray("authors") ?: JSONArray()).requireAtMost(MAX_PUBLICATION_AUTHORS, "Series authors")
                        SeriesSummary(
                            libraryId = getString("library_id"),
                            name = getString("name"),
                            authors = List(authorValues.length()) { authorValues.getString(it) },
                            count = optInt("count"),
                            coverPath = optString("cover_url").takeIf(String::isNotBlank),
                        )
                    }
                },
                nextCursor = response.nextCursor,
            )
        }
    }

    fun authors(credential: DeviceCredential, libraryId: String, cursor: String?, query: String = "", bucket: String = ""): AuthorPage {
        val requestQuery = buildString {
            append("?library=").append(encode(libraryId)).append("&limit=60")
            if (query.isNotBlank()) append("&q=").append(encode(query.trim()))
            if (bucket.isNotBlank()) append("&bucket=").append(encode(bucket))
            if (!cursor.isNullOrBlank()) append("&cursor=").append(encode(cursor))
        }
        return PerformanceDiagnostics.measure("network", "load_authors") { request(credential, "/api/android/v1/authors$requestQuery") }.let { response ->
            val values = JSONArray(response.body).requireAtMost(60, "Author page")
            AuthorPage(
                items = List(values.length()) { index -> values.getJSONObject(index).run { AuthorSummary(getString("library_id"), getString("name"), optInt("count"), optString("cover_url").takeIf(String::isNotBlank)) } },
                nextCursor = response.nextCursor,
            )
        }
    }

    fun coverUrl(credential: DeviceCredential, book: BookSummary): String =
        "${credential.serverUrl}/api/android/v1/covers/${encodePath(book.libraryId)}/${encodePath(book.id)}"

    /** The first cover of a collection, or null when it holds nothing with a cover. */
    fun collectionCoverUrl(credential: DeviceCredential, collection: PublicationCollection): String? =
        collection.covers.firstOrNull()?.substringAfter("/covers/", missingDelimiterValue = "")?.takeIf(String::isNotBlank)
            ?.let { "${credential.serverUrl}/api/android/v1/covers/$it" }

    fun seriesCoverUrl(credential: DeviceCredential, series: SeriesSummary): String? {
        val path = series.coverPath ?: return null
        val coverPath = path.substringAfter("/covers/", missingDelimiterValue = "")
        return coverPath.takeIf(String::isNotBlank)?.let { "${credential.serverUrl}/api/android/v1/covers/$it" }
    }

    fun authorCoverUrl(credential: DeviceCredential, author: AuthorSummary): String? {
        val path = author.coverPath ?: return null
        val coverPath = path.substringAfter("/covers/", missingDelimiterValue = "")
        return coverPath.takeIf(String::isNotBlank)?.let { "${credential.serverUrl}/api/android/v1/covers/$it" }
    }

    /**
     * [textOnly] asks for an EPUB without its recorded narration, which the app streams instead: a read-along
     * book can be hundreds of megabytes of audio around a small amount of text. A book without narration is sent whole.
     */
    fun publicationUrl(credential: DeviceCredential, book: BookSummary, textOnly: Boolean = false): String =
        "${credential.serverUrl}/api/android/v1/publications/${encodePath(book.libraryId)}/${encodePath(book.id)}" + if (textOnly) "?text=1" else ""

    /**
     * A `{pageNumber}` template for [book]'s OPDS-PSE comic pages, resolved against
     * [DeviceCredential.serverUrl] by [net.kdhmedia.mochil.reader.comic.StreamedComicPageSource].
     * This is the Bearer-gated Android route (mirroring [publicationUrl]'s pattern), never the
     * server's raw `page_stream_url` value: that field only gates `/stream/{id}`, which accepts a
     * session cookie or HTTP Basic auth but never a device Bearer token, so trusting it directly
     * would 401 on every real device.
     */
    fun publicationPageTemplate(book: BookSummary): String =
        "/api/android/v1/publications/${encodePath(book.libraryId)}/${encodePath(book.id)}/pages/{pageNumber}"

    private fun request(credential: DeviceCredential, path: String): CatalogueResponse {
        val connection = openAndroidApiGet(credential, path)
        return connection.run {
            try {
                if (responseCode !in 200..299) {
                    val message = errorStream?.use { it.readBoundedResponseText(MAX_ERROR_RESPONSE_BYTES) }.orEmpty()
                    throw CatalogueException(responseCode, message.ifBlank { "The server returned $responseCode." })
                }
                CatalogueResponse(
                    body = inputStream.use { it.readBoundedResponseText() },
                    nextCursor = getHeaderField("X-Next-Cursor")?.takeIf { it.isNotBlank() },
                )
            } finally {
                disconnect()
            }
        }
    }

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString())
    private fun encodePath(value: String) = encode(value).replace("+", "%20")
}

internal fun booksRequestQuery(libraryId: String, series: String?, cursor: String?, query: String, bucket: String, sort: String, status: String = ""): String =
    groupedBooksRequestQuery(libraryId, series, null, cursor, query, bucket, sort, 60, status)

private fun groupedBooksRequestQuery(libraryId: String, series: String?, author: String?, cursor: String?, query: String, bucket: String, sort: String, limit: Int = 60, status: String = ""): String = buildString {
    fun encoded(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString())
    append("?library=").append(encoded(libraryId)).append("&limit=").append(limit.coerceIn(1, 500))
    if (series != null) append("&series=").append(encoded(series))
    if (author != null) append("&author=").append(encoded(author))
    if (query.isNotBlank()) append("&q=").append(encoded(query.trim()))
    if (bucket.isNotBlank()) append("&bucket=").append(encoded(bucket))
    if (sort.isNotBlank()) append("&sort=").append(encoded(sort))
    // Only a value the server knows is sent; the filter is part of the shelf's identity, so it goes before the cursor that pages through it.
    BookStatusFilter.fromWire(status).wire.takeIf(String::isNotEmpty)?.let { append("&status=").append(it) }
    if (!cursor.isNullOrBlank()) append("&cursor=").append(encoded(cursor))
}

private data class CatalogueResponse(val body: String, val nextCursor: String?)

private fun CatalogueResponse.asBookPage(): BookPage {
    val values = JSONArray(body).requireAtMost(500, "Grouped book page")
    return BookPage(
        items = List(values.length()) { index -> values.getJSONObject(index).run {
            val authorValues = (optJSONArray("authors") ?: JSONArray()).requireAtMost(MAX_PUBLICATION_AUTHORS, "Publication authors")
            BookSummary(getString("id"), getString("library_id"), optString("title", "Untitled"), List(authorValues.length()) { authorValues.getString(it) }, optString("series"), if (has("series_index") && !isNull("series_index")) optDouble("series_index") else null, optString("format"), optString("media_type", "application/octet-stream"), optString("language"), optString("description"), optLong("size"))
        } },
        nextCursor = nextCursor,
    )
}

class CatalogueException(val status: Int, message: String) : Exception(message)

private const val MAX_LIBRARIES = 5_000
