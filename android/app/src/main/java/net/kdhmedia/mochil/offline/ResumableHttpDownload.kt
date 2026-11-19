package net.kdhmedia.mochil.offline

import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.net.HttpURLConnection
import org.json.JSONObject
import net.kdhmedia.mochil.network.readBoundedResponseText

data class DownloadResumeRequest(val offset: Long = 0, val validator: String = "")

internal enum class DownloadResponseMode { Restart, Append }

internal fun downloadResponseMode(responseCode: Int, requestedOffset: Long, contentRange: String?): DownloadResponseMode {
    if (requestedOffset <= 0 || responseCode != HttpURLConnection.HTTP_PARTIAL) return DownloadResponseMode.Restart
    val start = contentRange?.substringBefore('/')?.substringAfter("bytes", "")?.substringBefore('-')?.trim()?.toLongOrNull()
    if (start != requestedOffset) throw IOException("The server returned an invalid partial download range.")
    return DownloadResponseMode.Append
}

/** Owns validator-safe HTTP range continuation and bounded partial-file state. */
object ResumableHttpDownload {
    const val HTTP_RANGE_NOT_SATISFIABLE = 416
    private const val MAX_METADATA_BYTES = 4096

    fun configure(connection: HttpURLConnection, partial: File): DownloadResumeRequest {
        val saved = readMetadata(partial)
        if (!partial.isFile || partial.length() <= 0 || saved == null) {
            discard(partial)
            return DownloadResumeRequest()
        }
        return DownloadResumeRequest(partial.length(), saved.validator).also { request ->
            connection.setRequestProperty("Range", "bytes=${request.offset}-")
            connection.setRequestProperty("If-Range", request.validator)
        }
    }

    fun copyResponse(
        connection: HttpURLConnection,
        partial: File,
        directory: File,
        maximumBytes: Long,
        request: DownloadResumeRequest,
        onProgress: (Long) -> Unit = {},
    ): Long {
        val mode = downloadResponseMode(connection.responseCode, request.offset, connection.getHeaderField("Content-Range"))
        val offset = if (mode == DownloadResponseMode.Append) request.offset else 0L
        val responseBytes = connection.contentLengthLong
        val expectedTotal = contentRangeTotal(connection.getHeaderField("Content-Range"))
            ?: responseBytes.takeIf { it > 0 }?.plus(offset)
        if (expectedTotal != null && expectedTotal > maximumBytes) throw IOException("This file is too large for offline download.")
        DownloadSpacePolicy.requireCapacity(directory, responseBytes, (maximumBytes - offset).coerceAtLeast(0))

        val validator = responseValidator(connection).ifBlank { request.validator.takeIf { mode == DownloadResponseMode.Append }.orEmpty() }
        if (validator.isNotBlank()) writeMetadata(partial, validator)
        else metadataFile(partial).delete()

        try {
            connection.inputStream.use { input ->
                FileOutputStream(partial, mode == DownloadResponseMode.Append).buffered().use { output ->
                    DownloadSpacePolicy.copyBounded(input, output, directory, maximumBytes - offset) { onProgress(offset + it) }
                }
            }
            if (partial.length() <= 0) throw IOException("The downloaded file is empty.")
            if (expectedTotal != null && partial.length() != expectedTotal) throw IOException("The download ended before the complete file arrived.")
            return partial.length()
        } catch (failure: Throwable) {
            if (validator.isBlank()) discard(partial)
            throw failure
        }
    }

    fun finish(partial: File) {
        metadataFile(partial).delete()
    }

    fun discard(partial: File) {
        partial.delete()
        metadataFile(partial).delete()
    }

    internal fun isRecoverable(partial: File): Boolean = partial.isFile && partial.length() > 0 && readMetadata(partial) != null

    internal fun metadataFile(partial: File) = File(partial.parentFile, partial.name + ".resume")

    private fun responseValidator(connection: HttpURLConnection): String {
        val etag = connection.getHeaderField("ETag")?.trim().orEmpty()
        if (etag.isNotBlank() && !etag.startsWith("W/", ignoreCase = true)) return etag
        return connection.getHeaderField("Last-Modified")?.trim().orEmpty()
    }

    private fun contentRangeTotal(value: String?): Long? = value?.substringAfter('/', "")?.trim()?.takeUnless { it == "*" }?.toLongOrNull()

    private fun writeMetadata(partial: File, validator: String) {
        metadataFile(partial).writeText(JSONObject().put("validator", validator).toString())
    }

    private fun readMetadata(partial: File): DownloadResumeMetadata? = runCatching {
        val validator = JSONObject(metadataFile(partial).inputStream().use { it.readBoundedResponseText(MAX_METADATA_BYTES) }).optString("validator").trim()
        validator.takeIf(String::isNotBlank)?.let(::DownloadResumeMetadata)
    }.getOrNull()

    private data class DownloadResumeMetadata(val validator: String)

}
