package net.kdhmedia.mochil.catalogue

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject

/** The reader's own label for a book, in the order the book menu lists them. */
internal enum class ReadingStatus(val wire: String, val label: String) {
    Reading("reading", "Reading"),
    Read("read", "Read"),
    OnHold("on_hold", "On hold"),
    Dropped("dropped", "Dropped"),
    Unread("unread", "Unread"),
}

internal fun BookSummary.readingStatus(): ReadingStatus = when {
    status == "on_hold" -> ReadingStatus.OnHold
    status == "dropped" -> ReadingStatus.Dropped
    finished -> ReadingStatus.Read
    (readingFraction ?: 0.0) > 0.0 -> ReadingStatus.Reading
    else -> ReadingStatus.Unread
}

/** The book as the list will show it once the server has accepted [status]. */
internal fun BookSummary.withReadingStatus(status: ReadingStatus): BookSummary = when (status) {
    ReadingStatus.Read -> copy(finished = true, status = "", readingFraction = 1.0)
    ReadingStatus.Reading -> copy(finished = false, status = "", readingFraction = (readingFraction ?: 0.0).coerceAtLeast(0.001))
    ReadingStatus.OnHold, ReadingStatus.Dropped -> copy(finished = false, status = status.wire)
    ReadingStatus.Unread -> copy(finished = false, status = "", readingFraction = null)
}

internal fun readingStatusMessage(status: ReadingStatus) = when (status) {
    ReadingStatus.Read -> "Marked as read."
    ReadingStatus.Reading -> "Back in your reading list."
    ReadingStatus.OnHold -> "Put on hold. It stays out of Continue reading until you resume it."
    ReadingStatus.Dropped -> "Marked as dropped. It stays out of Continue reading until you resume it."
    ReadingStatus.Unread -> "Marked as unread."
}

internal class ReadingStatusClient {
    fun set(credential: DeviceCredential, libraryId: String, bookId: String, status: String) {
        val path = "/api/android/v1/reading/${encode(libraryId)}/${encode(bookId)}/status"
        val connection = URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection
        connection.requestMethod = "POST"
        connection.connectTimeout = 10_000
        connection.readTimeout = 20_000
        connection.doOutput = true
        connection.instanceFollowRedirects = false
        connection.setRequestProperty("Authorization", "Bearer ${credential.token}")
        connection.setRequestProperty("Content-Type", "application/json")
        try {
            connection.outputStream.use { it.write(JSONObject().put("status", status).toString().toByteArray()) }
            val responseStatus = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(responseStatus)
            if (responseStatus !in 200..299) error("The server returned $responseStatus.")
        } finally { connection.disconnect() }
    }

    private fun encode(value: String) = URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
}
