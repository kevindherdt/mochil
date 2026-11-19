package net.kdhmedia.mochil.webdav

import java.io.ByteArrayInputStream
import java.net.URI
import java.net.URLDecoder
import java.nio.charset.StandardCharsets
import javax.xml.parsers.DocumentBuilderFactory
import org.w3c.dom.Element

object WebDavParser {
    fun parse(bytes: ByteArray, requestUrl: String): WebDavPage {
        require(bytes.size in 1..MAX_RESPONSE_BYTES) { "The WebDAV response is empty or too large." }
        val safeBytes = withoutHarmlessDoctype(bytes)
        val root = try {
            val factory = secureDocumentBuilderFactory()
            factory.newDocumentBuilder().parse(ByteArrayInputStream(safeBytes)).documentElement
        } catch (error: WebDavRequestException) {
            throw error
        } catch (_: Exception) {
            throw WebDavRequestException("response", "This WebDAV server returned an unreadable folder response.")
        }
        require(root.localName == "multistatus") { "This address did not return a WebDAV folder." }
        val request = URI(requestUrl)
        val requestPath = normalizedPath(request.path)
        val entries = root.children("response").take(MAX_ENTRIES).mapNotNull { response ->
            val hrefText = response.text("href").take(MAX_URL)
            val resolved = runCatching { request.resolve(hrefText) }.getOrNull() ?: return@mapNotNull null
            if (!sameOrigin(request, resolved) || normalizedPath(resolved.path) == requestPath) return@mapNotNull null
            val successful = response.children("propstat").firstOrNull { it.text("status").contains(" 200 ") } ?: return@mapNotNull null
            val property = successful.children("prop").firstOrNull() ?: return@mapNotNull null
            val collection = property.children("resourcetype").firstOrNull()?.children("collection")?.isNotEmpty() == true
            val name = property.text("displayname").ifBlank { decodedName(resolved.path) }.take(MAX_NAME)
            WebDavEntry(
                href = resolved.toString(),
                name = name.ifBlank { "Untitled" },
                collection = collection,
                contentLength = property.text("getcontentlength").toLongOrNull()?.coerceAtLeast(0) ?: 0,
                contentType = property.text("getcontenttype").substringBefore(';').trim().take(MAX_CONTENT_TYPE),
                etag = property.text("getetag").take(MAX_ETAG),
                modified = property.text("getlastmodified").take(MAX_MODIFIED),
            )
        }.filter(WebDavEntry::supported).distinctBy(WebDavEntry::href)
            .sortedWith(compareByDescending<WebDavEntry> { it.collection }.thenBy { it.name.lowercase() })
        return WebDavPage(request.toString(), entries)
    }

    private fun decodedName(path: String): String = URLDecoder.decode(path.trimEnd('/').substringAfterLast('/'), StandardCharsets.UTF_8.name())
    private fun normalizedPath(path: String) = "/" + path.trim('/').let { if (it.isBlank()) "" else "$it/" }
    private fun sameOrigin(left: URI, right: URI) = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && port(left) == port(right)
    private fun port(uri: URI) = if (uri.port >= 0) uri.port else 443
    private fun Element.text(name: String): String = children(name).firstOrNull()?.textContent?.trim().orEmpty()
    private fun Element.children(name: String): List<Element> = buildList {
        for (index in 0 until childNodes.length) (childNodes.item(index) as? Element)?.takeIf { it.localName == name }?.let(::add)
    }

    const val MAX_RESPONSE_BYTES = 4 * 1024 * 1024
    private const val MAX_ENTRIES = 2_000
    private const val MAX_URL = 4_096
    private const val MAX_NAME = 500
    private const val MAX_CONTENT_TYPE = 160
    private const val MAX_ETAG = 500
    private const val MAX_MODIFIED = 160
}

/** Vendor parsers do not all expose the same optional JAXP switches. The
 * byte-level declaration filter remains the primary, fail-closed DTD defence. */
private fun secureDocumentBuilderFactory(): DocumentBuilderFactory = DocumentBuilderFactory.newInstance().apply {
    isNamespaceAware = true
    runCatching { isXIncludeAware = false }
    isExpandEntityReferences = false
    runCatching { setFeature("http://apache.org/xml/features/disallow-doctype-decl", true) }
    runCatching { setFeature("http://xml.org/sax/features/external-general-entities", false) }
    runCatching { setFeature("http://xml.org/sax/features/external-parameter-entities", false) }
    runCatching { setAttribute("http://javax.xml.XMLConstants/property/accessExternalDTD", "") }
    runCatching { setAttribute("http://javax.xml.XMLConstants/property/accessExternalSchema", "") }
}

/**
 * Some Apache-compatible servers emit a name-only DOCTYPE. It carries no
 * entities and is safe to discard. External identifiers and internal subsets
 * remain rejected before the hardened XML parser sees the document.
 */
internal fun withoutHarmlessDoctype(bytes: ByteArray): ByteArray {
    val marker = "<!DOCTYPE".encodeToByteArray()
    val start = bytes.indexOfAsciiIgnoreCase(marker)
    if (start < 0) return bytes
    var quote = 0
    var end = -1
    for (index in start + marker.size until minOf(bytes.size, start + MAX_DOCTYPE_BYTES)) {
        val value = bytes[index].toInt() and 0xff
        if (quote != 0) {
            if (value == quote) quote = 0
            continue
        }
        if (value == '"'.code || value == '\''.code) quote = value
        else if (value == '['.code) throw WebDavRequestException("response", "This WebDAV response contains an unsafe document declaration.")
        else if (value == '>'.code) { end = index; break }
    }
    if (end < 0) throw WebDavRequestException("response", "This WebDAV response contains an invalid document declaration.")
    val declaration = bytes.copyOfRange(start, end + 1).decodeToString().uppercase()
    if ("SYSTEM" in declaration || "PUBLIC" in declaration) throw WebDavRequestException("response", "This WebDAV response references an external document declaration.")
    val stripped = bytes.copyOfRange(0, start) + bytes.copyOfRange(end + 1, bytes.size)
    if (stripped.indexOfAsciiIgnoreCase(marker) >= 0) {
        throw WebDavRequestException("response", "This WebDAV response contains multiple document declarations.")
    }
    return stripped
}

private fun ByteArray.indexOfAsciiIgnoreCase(marker: ByteArray): Int {
    outer@ for (index in 0..size - marker.size) {
        for (offset in marker.indices) {
            val left = this[index + offset].toInt().toChar().uppercaseChar()
            val right = marker[offset].toInt().toChar().uppercaseChar()
            if (left != right) continue@outer
        }
        return index
    }
    return -1
}

private const val MAX_DOCTYPE_BYTES = 4_096
