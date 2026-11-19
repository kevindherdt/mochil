@file:androidx.annotation.OptIn(markerClass = [androidx.media3.common.util.UnstableApi::class])

package net.kdhmedia.mochil.webdav

import android.content.Context
import android.net.Uri
import androidx.media3.common.C
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.BaseDataSource
import androidx.media3.datasource.DataSource
import androidx.media3.datasource.DataSpec
import androidx.media3.datasource.DefaultDataSource
import androidx.media3.datasource.TransferListener
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URI

@UnstableApi
class BoboMediaDataSourceFactory(
    context: Context,
    serverHttpFactory: DataSource.Factory,
) : DataSource.Factory {
    private val applicationContext = context.applicationContext
    private val defaultFactory = DefaultDataSource.Factory(applicationContext, serverHttpFactory)
    override fun createDataSource(): DataSource = RoutingMediaDataSource(defaultFactory.createDataSource(), WebDavMediaDataSource(applicationContext))
}

@UnstableApi
private class RoutingMediaDataSource(private val ordinary: DataSource, private val webDav: DataSource) : DataSource {
    private var active: DataSource? = null
    override fun open(dataSpec: DataSpec): Long {
        check(active == null) { "A media source is already open." }
        val selected = if (dataSpec.uri.scheme == WEB_DAV_STREAM_SCHEME) webDav else ordinary
        active = selected
        return try { selected.open(dataSpec) } catch (failure: Throwable) { runCatching { selected.close() }; active = null; throw failure }
    }
    override fun read(buffer: ByteArray, offset: Int, length: Int): Int = requireNotNull(active).read(buffer, offset, length)
    override fun getUri(): Uri? = active?.uri
    override fun getResponseHeaders(): Map<String, List<String>> = active?.responseHeaders.orEmpty()
    override fun close() { try { active?.close() } finally { active = null } }
    override fun addTransferListener(transferListener: TransferListener) { ordinary.addTransferListener(transferListener); webDav.addTransferListener(transferListener) }
}

@UnstableApi
internal class WebDavMediaDataSource(context: Context) : BaseDataSource(true) {
    private val references = WebDavStreamReferenceStore(context)
    private val connections = WebDavConnectionStore(context)
    private var request: HttpURLConnection? = null
    private var input: InputStream? = null
    private var openedUri: Uri? = null
    private var remaining = C.LENGTH_UNSET.toLong()
    private var headers: Map<String, List<String>> = emptyMap()
    private var transferOpen = false

    override fun open(dataSpec: DataSpec): Long {
        transferInitializing(dataSpec)
        val token = parseWebDavStreamToken(dataSpec.uri) ?: throw WebDavRequestException("reference", "This WebDAV stream reference is invalid.")
        val reference = references.resolve(token) ?: throw WebDavRequestException("reference", "This WebDAV stream is no longer available. Open it from the library again.")
        val connection = connections.list().firstOrNull { it.id == reference.connectionId }
            ?: throw WebDavRequestException("reference", "This WebDAV connection was removed.")
        if (reference.credentialKey != webDavCredentialKey(connection)) throw WebDavRequestException("reference", "This WebDAV connection changed. Open the item from the library again.")
        val root = validatedWebDavUrl(connection.rootUrl)
        var target = validatedWebDavUrl(reference.href)
        repeat(MAX_REDIRECTS + 1) { redirect ->
            if (!sameWebDavOrigin(root, target)) throw WebDavRequestException("redirect", "The WebDAV stream left the configured server.")
            val response = (target.toURL().openConnection() as HttpURLConnection).apply {
                instanceFollowRedirects = false; connectTimeout = 15_000; readTimeout = 30_000
                setRequestProperty("Accept-Encoding", "identity")
                if (connection.username.isNotBlank()) setRequestProperty("Authorization", basicAuthorization(connection.username, connection.password))
                rangeHeader(dataSpec)?.let { setRequestProperty("Range", it) }
            }
            val status = response.responseCode
            if (status in 300..399) {
                if (redirect >= MAX_REDIRECTS) throw WebDavRequestException("redirect", "The WebDAV stream returned too many redirects.")
                val location = response.getHeaderField("Location") ?: throw WebDavRequestException("redirect", "The WebDAV stream returned an invalid redirect.")
                response.disconnect()
                target = validatedWebDavUrl(target.resolve(location).toString())
                return@repeat
            }
            if (status == 416) { response.disconnect(); throw WebDavRequestException("range", "The requested WebDAV playback position is outside this file.") }
            if (status !in 200..299) { response.disconnect(); throw webDavStatusFailure(status) }
            if (dataSpec.position > 0 && status != HttpURLConnection.HTTP_PARTIAL) { response.disconnect(); throw WebDavRequestException("range", "This WebDAV server does not support seeking for this file.") }
            request = response
            openedUri = Uri.parse(target.toString())
            headers = response.headerFields
            remaining = when {
                dataSpec.length != C.LENGTH_UNSET.toLong() -> dataSpec.length
                response.contentLengthLong >= 0 -> response.contentLengthLong
                else -> C.LENGTH_UNSET.toLong()
            }
            input = try { response.inputStream } catch (failure: Throwable) { request = null; response.disconnect(); throw failure }
            transferStarted(dataSpec)
            transferOpen = true
            return remaining
        }
        throw WebDavRequestException("redirect", "The WebDAV stream could not be opened.")
    }

    override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
        if (length == 0) return 0
        if (remaining == 0L) return C.RESULT_END_OF_INPUT
        val requested = if (remaining == C.LENGTH_UNSET.toLong()) length else minOf(length.toLong(), remaining).toInt()
        val count = input?.read(buffer, offset, requested) ?: return C.RESULT_END_OF_INPUT
        if (count < 0) return C.RESULT_END_OF_INPUT
        if (remaining != C.LENGTH_UNSET.toLong()) remaining -= count
        bytesTransferred(count)
        return count
    }

    override fun getUri(): Uri? = openedUri
    override fun getResponseHeaders(): Map<String, List<String>> = headers
    override fun close() {
        try { input?.close() } finally {
            input = null; request?.disconnect(); request = null; openedUri = null; headers = emptyMap(); remaining = C.LENGTH_UNSET.toLong()
            if (transferOpen) { transferOpen = false; transferEnded() }
        }
    }

    private companion object { const val MAX_REDIRECTS = 3 }
}

internal fun parseWebDavStreamToken(uri: Uri): String? = uri.takeIf { it.scheme == WEB_DAV_STREAM_SCHEME && it.host == "stream" && it.pathSegments.size == 1 }
    ?.pathSegments?.singleOrNull()?.takeIf { it.matches(Regex("^[a-f0-9]{36}$")) }

internal fun rangeHeader(dataSpec: DataSpec): String? {
    return buildWebDavRangeHeader(dataSpec.position, dataSpec.length)
}

internal fun buildWebDavRangeHeader(position: Long, length: Long): String? {
    if (position == 0L && length == C.LENGTH_UNSET.toLong()) return null
    val end = length.takeIf { it != C.LENGTH_UNSET.toLong() }?.let { position + it - 1 }
    return "bytes=$position-${end ?: ""}"
}

private fun sameWebDavOrigin(left: URI, right: URI): Boolean = left.scheme.equals(right.scheme, true) && left.host.equals(right.host, true) &&
    (if (left.port >= 0) left.port else 443) == (if (right.port >= 0) right.port else 443)
