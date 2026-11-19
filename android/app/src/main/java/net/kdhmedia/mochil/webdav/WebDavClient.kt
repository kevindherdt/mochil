package net.kdhmedia.mochil.webdav

import java.net.URI
import java.nio.charset.StandardCharsets
import java.util.Base64
import java.util.concurrent.TimeUnit
import net.kdhmedia.mochil.opds.readBounded
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

class WebDavClient(private val connection: WebDavConnection) {
    fun list(url: String = connection.rootUrl, validator: String = ""): WebDavPage {
        val root = validatedWebDavUrl(connection.rootUrl)
        var target = validatedWebDavUrl(url)
        repeat(MAX_REDIRECTS + 1) { redirect ->
            if (!sameOrigin(root, target)) throw WebDavRequestException("redirect", "WebDAV navigation left the configured server.")
            val request = Request.Builder().url(target.toString()).method("PROPFIND", PROPFIND_BODY.toRequestBody(XML_MEDIA_TYPE))
                .header("Depth", "1").header("Accept", "application/xml, text/xml").apply {
                    if (validator.isNotBlank()) header("If-None-Match", validator)
                    if (connection.username.isNotBlank()) header("Authorization", basicAuthorization(connection.username, connection.password))
                }.build()
            HTTP.newCall(request).execute().use { response ->
                if (response.code in 300..399) {
                    if (redirect >= MAX_REDIRECTS) throw WebDavRequestException("redirect", "The WebDAV server returned too many redirects.")
                    target = validatedWebDavUrl(target.resolve(response.header("Location") ?: error("The WebDAV server returned an invalid redirect.")).toString())
                    return@repeat
                }
                if (response.code == 304) return WebDavPage(target.toString(), emptyList(), validator, notModified = true)
                if (response.code != 207 && !response.isSuccessful) throw webDavStatusFailure(response.code)
                val body = response.body
                return WebDavParser.parse(readBounded(body.byteStream(), WebDavParser.MAX_RESPONSE_BYTES), target.toString())
                    .copy(validator = response.header("ETag").orEmpty().take(500))
            }
        }
        error("The WebDAV request could not be completed.")
    }

    fun supportsRanges(url: String): Boolean {
        val root = validatedWebDavUrl(connection.rootUrl)
        var target = validatedWebDavUrl(url)
        repeat(MAX_REDIRECTS + 1) { redirect ->
            if (!sameOrigin(root, target)) throw WebDavRequestException("redirect", "WebDAV playback left the configured server.")
            val request = Request.Builder().url(target.toString()).get().header("Range", "bytes=0-0").header("Accept-Encoding", "identity").apply {
                if (connection.username.isNotBlank()) header("Authorization", basicAuthorization(connection.username, connection.password))
            }.build()
            HTTP.newCall(request).execute().use { response ->
                if (response.code in 300..399) {
                    if (redirect >= MAX_REDIRECTS) throw WebDavRequestException("redirect", "The WebDAV server returned too many redirects.")
                    target = validatedWebDavUrl(target.resolve(response.header("Location") ?: throw WebDavRequestException("redirect", "The WebDAV server returned an invalid redirect.")).toString())
                    return@repeat
                }
                if (response.code == 200) return false
                if (response.code != 206) throw webDavStatusFailure(response.code)
                return response.header("Content-Range").orEmpty().startsWith("bytes 0-0/")
            }
        }
        throw WebDavRequestException("redirect", "The WebDAV range check could not be completed.")
    }

    private companion object {
        const val MAX_REDIRECTS = 3
        val XML_MEDIA_TYPE = "application/xml; charset=utf-8".toMediaType()
        val PROPFIND_BODY = """<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:"><d:prop><d:displayname/><d:resourcetype/><d:getcontentlength/><d:getcontenttype/><d:getetag/><d:getlastmodified/></d:prop></d:propfind>"""
        val HTTP = OkHttpClient.Builder().followRedirects(false).followSslRedirects(false).connectTimeout(15, TimeUnit.SECONDS).readTimeout(30, TimeUnit.SECONDS).build()
    }
}

internal fun basicAuthorization(username: String, password: String): String = "Basic " + Base64.getEncoder()
    .encodeToString("$username:$password".toByteArray(StandardCharsets.UTF_8))

internal fun webDavStatusMessage(status: Int): String = when (status) {
    401, 403 -> "The WebDAV server rejected these credentials."
    404 -> "The WebDAV folder was not found. Check the complete server path."
    405 -> "This address does not allow WebDAV folder browsing. Check the provider's WebDAV endpoint."
    else -> "The WebDAV server returned HTTP $status."
}

internal class WebDavRequestException(val category: String, message: String) : java.io.IOException(message)
internal fun webDavStatusFailure(status: Int) = WebDavRequestException(when (status) {
    401, 403 -> "auth"; 404 -> "not_found"; 405 -> "endpoint"; else -> "server"
}, webDavStatusMessage(status))

internal fun webDavDiagnosticName(error: Throwable): String = when ((error as? WebDavRequestException)?.category) {
    "auth" -> "webdav_auth_error"; "not_found" -> "webdav_not_found_error"; "endpoint" -> "webdav_endpoint_error"
    "redirect" -> "webdav_redirect_error"; "server" -> "webdav_server_error"; "range" -> "webdav_range_error"
    "reference" -> "webdav_reference_error"; "response" -> "webdav_response_error"; else -> "webdav_transport_error"
}

private fun sameOrigin(left: URI, right: URI) = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) && port(left) == port(right)
private fun port(uri: URI) = if (uri.port >= 0) uri.port else 443
