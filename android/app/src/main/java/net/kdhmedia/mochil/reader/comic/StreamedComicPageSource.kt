package net.kdhmedia.mochil.reader.comic

import java.io.ByteArrayOutputStream
import java.net.HttpURLConnection
import java.net.URI

/**
 * Fetches CBZ/manga pages one at a time over HTTP instead of downloading the whole archive,
 * using the Bearer-gated Android publications route (`.../api/android/v1/publications/{library}/
 * {id}/pages/{pageNumber}`, built by [net.kdhmedia.mochil.catalogue.CatalogueClient.publicationPageTemplate]).
 * That route authorizes the same way [net.kdhmedia.mochil.catalogue.PublicationOpener]'s full
 * downloads do; the server's raw `page_stream_url` field (which gates `/stream/{id}`, a
 * session/Basic-only endpoint) is only ever used as a streamability flag, never dereferenced.
 *
 * Mirrors [net.kdhmedia.mochil.catalogue.PublicationOpener]'s book/cover download
 * authentication: the device bearer token is attached only when the request targets the same
 * origin as [serverUrl], never on a cross-origin redirect target. Redirects are not followed at
 * all, since the page URL is always a server-relative path resolved against [serverUrl]. This
 * same-origin check is kept as a defensive safety net even though the route is now built
 * entirely from [serverUrl] itself, so it is same-origin by construction.
 */
class StreamedComicPageSource(
    private val serverUrl: String,
    private val pageStreamURL: String,
    override val pageCount: Int,
    private val token: String?,
    private val maxAttempts: Int = 3,
    private val connectTimeoutMillis: Int = 15_000,
    private val readTimeoutMillis: Int = 30_000,
) : ComicPageSource {
    init {
        require(pageStreamURL.contains(PAGE_PLACEHOLDER)) { "This comic does not provide page streaming." }
    }

    override fun page(index: Int): ByteArray {
        require(index in 0 until pageCount) { "Requested comic page is out of range." }
        var lastError: Exception? = null
        repeat(maxAttempts) {
            try {
                return fetch(index)
            } catch (error: Exception) {
                lastError = error
            }
        }
        throw IllegalStateException("This comic page could not be loaded.", lastError)
    }

    override fun close() {
        // Nothing to release: every page fetch opens and closes its own connection.
    }

    private fun fetch(index: Int): ByteArray {
        val origin = URI(serverUrl)
        val target = origin.resolve(pageStreamURL.replace(PAGE_PLACEHOLDER, index.toString()))
        val connection = target.toURL().openConnection() as HttpURLConnection
        connection.instanceFollowRedirects = false
        connection.connectTimeout = connectTimeoutMillis
        connection.readTimeout = readTimeoutMillis
        connection.setRequestProperty("Accept", "image/*")
        if (!token.isNullOrBlank() && sameOrigin(origin, target)) {
            connection.setRequestProperty("Authorization", "Bearer $token")
        }
        try {
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode} while streaming this page.")
            val expected = connection.contentLengthLong
            if (expected > MAX_PAGE_BYTES) error("Comic page is too large.")
            return connection.inputStream.use { input ->
                val output = ByteArrayOutputStream(if (expected in 1..MAX_PAGE_BYTES) expected.toInt() else 64 * 1024)
                val buffer = ByteArray(32 * 1024)
                var total = 0L
                while (true) {
                    val read = input.read(buffer)
                    if (read < 0) break
                    total += read
                    if (total > MAX_PAGE_BYTES) error("Comic page is too large.")
                    output.write(buffer, 0, read)
                }
                output.toByteArray()
            }
        } finally {
            connection.disconnect()
        }
    }

    private fun sameOrigin(left: URI, right: URI): Boolean =
        left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && effectivePort(left) == effectivePort(right)

    private fun effectivePort(uri: URI): Int = when {
        uri.port >= 0 -> uri.port
        uri.scheme.equals("https", true) -> 443
        else -> 80
    }

    private companion object {
        const val PAGE_PLACEHOLDER = "{pageNumber}"
        const val MAX_PAGE_BYTES = 64L shl 20
    }
}
