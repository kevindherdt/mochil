package net.kdhmedia.mochil.opds

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import java.util.Base64
import java.io.ByteArrayOutputStream
import java.io.InputStream
import net.kdhmedia.mochil.source.CatalogueSource
import net.kdhmedia.mochil.source.SourceItem
import net.kdhmedia.mochil.source.SourceLibrary

data class OpdsConnection(
    val id: String,
    val name: String,
    val catalogueUrl: String,
    val username: String = "",
    val password: String = "",
    val cacheGeneration: String = id,
)
data class OpdsPage(val title: String, val items: List<SourceItem>, val nextUrl: String?, val searchAvailable: Boolean)
private data class OpdsResponse(val bytes: ByteArray, val url: URI, val contentType: String)

class OpdsCatalogueSource(private val connection: OpdsConnection) : CatalogueSource {
    override val id: String = "opds:${connection.id}"
    private var rootFeed: OpdsFeed? = null

    override suspend fun libraries(): List<SourceLibrary> {
        val feed = fetch(connection.catalogueUrl)
        rootFeed = feed
        return listOf(SourceLibrary(id, connection.name.ifBlank { feed.title }, "publications"))
    }

    override suspend fun latest(libraryId: String, cursor: String?): List<SourceItem> = page(cursor).items

    override suspend fun search(libraryId: String, query: String, cursor: String?): List<SourceItem> {
        return searchPage(query, cursor).items
    }

    fun page(url: String? = null): OpdsPage = fetch(url ?: connection.catalogueUrl).asPage()
    fun searchPage(query: String, url: String? = null): OpdsPage = fetch(url ?: searchUrl(query)).asPage()

    private fun searchUrl(query: String): String {
        val feed = rootFeed ?: fetch(connection.catalogueUrl).also { rootFeed = it }
        val advertised = feed.searchLink?.href ?: error("This OPDS catalogue does not advertise search.")
        val template = if (advertised.contains("{searchTerms")) advertised else {
            val response = fetchBytes(advertised)
            OpdsSearchParser.parseTemplate(response.bytes)
        }
        val encoded = URLEncoder.encode(query, StandardCharsets.UTF_8.name()).replace("+", "%20")
        val templateBase = if (advertised.contains("{searchTerms")) connection.catalogueUrl else advertised
        return resolve(templateBase, expandOpdsSearchTemplate(template, encoded))
    }

    private fun fetch(initialUrl: String): OpdsFeed {
        val response = fetchBytes(initialUrl)
        val feed = if (response.contentType.substringBefore(';').trim().equals("application/opds+json", true) || response.bytes.firstOrNull { !it.toInt().toChar().isWhitespace() }?.toInt()?.toChar() == '{') Opds2Parser.parse(response.bytes) else OpdsParser.parse(response.bytes)
        return feed.resolveLinks(response.url)
    }

    private fun fetchBytes(initialUrl: String): OpdsResponse {
        val origin = validatedHttps(connection.catalogueUrl)
        var current = validatedHttps(resolve(connection.catalogueUrl, initialUrl))
        repeat(MAX_REDIRECTS + 1) { redirect ->
            val request = current.toURL().openConnection() as HttpURLConnection
            request.instanceFollowRedirects = false
            request.connectTimeout = 15_000
            request.readTimeout = 30_000
            request.setRequestProperty("Accept", "application/opds+json, application/atom+xml;profile=opds-catalog;q=0.9, application/atom+xml;q=0.8")
            if (sameOrigin(origin, current) && connection.username.isNotBlank()) {
                val token = Base64.getEncoder().encodeToString("${connection.username}:${connection.password}".toByteArray(StandardCharsets.UTF_8))
                request.setRequestProperty("Authorization", "Basic $token")
            }
            try {
                if (request.responseCode in 300..399) {
                    require(redirect < MAX_REDIRECTS) { "The OPDS server returned too many redirects." }
                    current = validatedHttps(current.resolve(request.getHeaderField("Location") ?: error("The OPDS server returned an invalid redirect.")).toString())
                    return@repeat
                }
                require(request.responseCode in 200..299) { "The OPDS server returned ${request.responseCode}." }
                val bytes = request.inputStream.use { input -> readBounded(input, OpdsParser.MAX_FEED_BYTES) }
                require(bytes.size <= OpdsParser.MAX_FEED_BYTES) { "The OPDS response is too large." }
                return OpdsResponse(bytes, current, request.contentType.orEmpty())
            } finally { request.disconnect() }
        }
        error("The OPDS request could not be completed.")
    }

    private fun OpdsFeed.asPage(): OpdsPage = OpdsPage(title, entries.mapNotNull { entry ->
        val acquisition = entry.acquisitionLink
        val navigation = entry.navigationLink
        if (acquisition == null && navigation == null) return@mapNotNull null
        SourceItem(entry.id, id, entry.title, entry.artworkLink?.href, entry.authors, acquisition?.mediaType.orEmpty(), acquisition?.href, navigation?.href)
    }, nextLink?.href, searchLink != null)

    private fun OpdsFeed.resolveLinks(base: URI) = copy(
        entries = entries.map { it.copy(links = it.links.map { link -> link.copy(href = resolve(base.toString(), link.href)) }) },
        navigation = navigation.map { it.copy(href = resolve(base.toString(), it.href)) },
    )
}

internal fun expandOpdsSearchTemplate(template: String, encodedQuery: String): String = template
    .replace("{searchTerms}", encodedQuery).replace("{searchTerms?}", encodedQuery)
    .replace("{?query}", "?query=$encodedQuery").replace("{?q}", "?q=$encodedQuery")

internal fun validatedHttps(value: String): URI = URI(value).also { uri ->
    require(uri.scheme.equals("https", true) && !uri.host.isNullOrBlank() && uri.userInfo == null) { "Use a credential-free HTTPS OPDS address." }
}
internal fun resolve(base: String, value: String): String {
    val escapedTemplate = value.replace("{", "%7B").replace("}", "%7D")
    return validatedHttps(URI(base).resolve(escapedTemplate).toString()).toString()
        .replace("%7B", "{").replace("%7D", "}")
}
private fun sameOrigin(left: URI, right: URI) = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && effectivePort(left) == effectivePort(right)
private fun effectivePort(uri: URI) = if (uri.port >= 0) uri.port else 443
private const val MAX_REDIRECTS = 3

internal fun readBounded(input: InputStream, maximumBytes: Int): ByteArray {
    require(maximumBytes > 0)
    val output = ByteArrayOutputStream(minOf(maximumBytes, 32 * 1024))
    val buffer = ByteArray(8 * 1024)
    var total = 0
    while (true) {
        val count = input.read(buffer)
        if (count < 0) return output.toByteArray()
        total += count
        require(total <= maximumBytes) { "The OPDS response is too large." }
        output.write(buffer, 0, count)
    }
}
