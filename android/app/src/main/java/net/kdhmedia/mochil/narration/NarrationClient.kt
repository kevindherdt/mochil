package net.kdhmedia.mochil.narration

import java.net.URLEncoder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.readBoundedResponseText
import org.json.JSONObject

/** Talks to the server's narration endpoints under /api/android/v1/narration/. */
class NarrationClient {
    /** The voices on offer, or null when narration is off or not set up on the server. */
    suspend fun status(credential: DeviceCredential): NarrationStatus? = withContext(Dispatchers.IO) {
        getJson(credential, "status")?.let(::parseNarrationStatus)
    }

    suspend fun chapters(credential: DeviceCredential, libraryId: String, bookId: String): List<NarrationChapter> = withContext(Dispatchers.IO) {
        getJson(credential, "book/${segment(libraryId)}/${segment(bookId)}")?.let(::parseNarrationChapters).orEmpty()
    }

    suspend fun sentences(credential: DeviceCredential, libraryId: String, bookId: String, chapter: Int): List<NarrationSentence> = withContext(Dispatchers.IO) {
        getJson(credential, "book/${segment(libraryId)}/${segment(bookId)}/chapters/$chapter")?.let(::parseNarrationSentences).orEmpty()
    }

    private fun getJson(credential: DeviceCredential, path: String): JSONObject? {
        val connection = openAndroidApiGet(credential, "/api/android/v1/narration/$path")
        return try {
            when (connection.responseCode) {
                404 -> null
                in 200..299 -> JSONObject(connection.inputStream.use { it.readBoundedResponseText() })
                else -> error("The server returned ${connection.responseCode} for narration.")
            }
        } finally { connection.disconnect() }
    }

    companion object {
        private fun segment(value: String) = URLEncoder.encode(value, "UTF-8").replace("+", "%20")

        /** The address of one sentence's audio. The player adds the device token as a header. */
        fun audioUrl(credential: DeviceCredential, libraryId: String, bookId: String, chapter: Int, index: Int, voice: String): String =
            credential.serverUrl.trimEnd('/') +
                "/api/android/v1/narration/book/${segment(libraryId)}/${segment(bookId)}/chapters/$chapter/audio/$index?voice=${segment(voice)}"
    }
}
