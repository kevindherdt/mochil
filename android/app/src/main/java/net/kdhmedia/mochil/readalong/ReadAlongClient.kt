package net.kdhmedia.mochil.readalong

import java.net.URLEncoder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.readBoundedResponseText
import org.json.JSONObject

/** Fetches the narration of a read-along book from the server, which streams the audio from the book. */
class ReadAlongClient {
    /** Null when the book has no recorded narration, or the server cannot say. */
    suspend fun overlay(credential: DeviceCredential, libraryId: String, bookId: String): ReadAlongOverlay? = withContext(Dispatchers.IO) {
        val connection = openAndroidApiGet(credential, "/api/android/v1/readium/${segment(libraryId)}/${segment(bookId)}/overlay.json")
        try {
            if (connection.responseCode !in 200..299) null
            else ReadAlongOverlay.parse(JSONObject(connection.inputStream.use { it.readBoundedResponseText(OVERLAY_LIMIT) }))
        } finally { connection.disconnect() }
    }

    companion object {
        private const val OVERLAY_LIMIT = 24 * 1024 * 1024
        private fun segment(value: String) = URLEncoder.encode(value, "UTF-8").replace("+", "%20")

        /** The address of one audio file inside the book. The player adds the device token as a header. */
        fun audioUrl(credential: DeviceCredential, libraryId: String, bookId: String, entry: String): String =
            credential.serverUrl.trimEnd('/') + "/api/android/v1/readium/${segment(libraryId)}/${segment(bookId)}/" +
                entry.split('/').joinToString("/") { segment(it) }
    }
}
