package net.kdhmedia.mochil.reader.epub

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.opds.readBounded
import org.json.JSONObject

internal class AnnotationNotesClient {
    suspend fun create(credential: DeviceCredential, libraryId: String, bookId: String): String = withContext(Dispatchers.IO) {
        require(libraryId.isNotBlank() && bookId.isNotBlank())
        val path = "/api/android/v1/annotations/${encode(libraryId)}/${encode(bookId)}/notes"
        val connection = (URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection).apply {
            requestMethod = "POST"
            doOutput = true
            connectTimeout = 10_000
            readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Content-Type", "application/json")
            setRequestProperty("Accept", "application/json")
            setFixedLengthStreamingMode(2)
        }
        try {
            connection.outputStream.use { it.write("{}".encodeToByteArray()) }
            val status = connection.responseCode
            if (status != HttpURLConnection.HTTP_CREATED) throw AnnotationNotesException(status)
            val bytes = connection.inputStream.use { readBounded(it, MAX_RESPONSE_BYTES) }
            JSONObject(bytes.decodeToString()).optString("title").take(MAX_TITLE_LENGTH).ifBlank { "Annotations" }
        } finally {
            connection.disconnect()
        }
    }

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")

    private companion object {
        const val MAX_RESPONSE_BYTES = 64 * 1024
        const val MAX_TITLE_LENGTH = 240
    }
}

internal class AnnotationNotesException(val status: Int) : Exception()

internal fun annotationNotesFailureMessage(error: Throwable): String = when ((error as? AnnotationNotesException)?.status) {
    401, 403 -> "Your session expired. Reconnect Mochil, then create the Notes page again."
    404 -> "This publication is no longer available on the server."
    409 -> "Resolve annotation conflicts or add a highlight before creating a Notes page."
    413 -> "These annotations are too large for one Notes page."
    503 -> "Mochil Notes is not enabled on this server."
    in 500..599 -> "The server could not create the Notes page. Try again shortly."
    else -> "Could not reach the server. Your highlights remain saved; try again when connected."
}
