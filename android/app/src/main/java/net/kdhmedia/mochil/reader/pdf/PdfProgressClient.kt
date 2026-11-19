package net.kdhmedia.mochil.reader.pdf

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.reader.currentReaderDeviceName
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.openAndroidApiGet
import org.json.JSONObject

data class PdfServerPosition(val revision: Long, val page: Int?, val fraction: Double)

enum class FixedLayoutLocatorKind(val wireName: String) {
    Pdf("pdf"),
    Comic("comic"),
}

sealed interface PdfSaveResult {
    data class Saved(val position: PdfServerPosition) : PdfSaveResult
    data class Conflict(val position: PdfServerPosition) : PdfSaveResult
}

class PdfProgressClient {
    suspend fun get(credential: DeviceCredential, libraryId: String, bookId: String, kind: FixedLayoutLocatorKind): PdfServerPosition? = withContext(Dispatchers.IO) {
        val connection = openAndroidApiGet(credential, path(libraryId, bookId))
        try {
            if (connection.responseCode == 404) return@withContext null
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode} while loading PDF progress.")
            decode(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }), kind)
        } finally { connection.disconnect() }
    }

    suspend fun put(credential: DeviceCredential, libraryId: String, bookId: String, page: Int, pageCount: Int, baseRevision: Long, kind: FixedLayoutLocatorKind): PdfSaveResult = withContext(Dispatchers.IO) {
        val boundedPage = page.coerceIn(0, (pageCount - 1).coerceAtLeast(0))
        val fraction = if (pageCount <= 1) 1.0 else boundedPage.toDouble() / (pageCount - 1)
        val locator = JSONObject().put("version", 1).put("page", boundedPage).put("page_count", pageCount.coerceAtLeast(1))
        val body = JSONObject().apply {
            put("fraction", fraction); put("base_revision", baseRevision); put("device_id", credential.deviceId); put("device_name", currentReaderDeviceName())
            put("locators", JSONObject().put(kind.wireName, locator.toString()))
        }.toString().encodeToByteArray()
        val connection = openMutation(credential, libraryId, bookId, "PUT").apply {
            doOutput = true; setRequestProperty("Content-Type", "application/json"); setFixedLengthStreamingMode(body.size)
        }
        try {
            connection.outputStream.use { it.write(body) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299 && status != 409) error("The server returned $status while saving PDF progress.")
            val stream = if (status == 409) connection.errorStream ?: connection.inputStream else connection.inputStream
            val position = decode(JSONObject(stream.use { it.readBoundedResponseText() }), kind)
            if (status == 409) PdfSaveResult.Conflict(position) else PdfSaveResult.Saved(position)
        } finally { connection.disconnect() }
    }

    private fun openMutation(credential: DeviceCredential, libraryId: String, bookId: String, method: String) =
        (URI("${credential.serverUrl}/api/android/v1/reading/${segment(libraryId)}/${segment(bookId)}").toURL().openConnection() as HttpURLConnection).apply {
            requestMethod = method; connectTimeout = 10_000; readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}"); setRequestProperty("Accept", "application/json")
        }

    private fun decode(value: JSONObject, kind: FixedLayoutLocatorKind): PdfServerPosition {
        val locators = value.optJSONObject("locators")
        val encoded = locators?.optString(kind.wireName)?.takeIf(String::isNotBlank)
            // Mochil versions before 0.16.44 wrote CBZ pages as PDF locators.
            ?: if (kind == FixedLayoutLocatorKind.Comic) locators?.optString(FixedLayoutLocatorKind.Pdf.wireName)?.takeIf(String::isNotBlank) else null
        val locator = encoded?.let { runCatching { JSONObject(it) }.getOrNull() }
        return PdfServerPosition(value.optLong("revision"), locator?.optInt("page")?.takeIf { it >= 0 }, value.optDouble("fraction").takeIf(Double::isFinite)?.coerceIn(0.0, 1.0) ?: 0.0)
    }

    private fun segment(value: String) = URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
    private fun path(libraryId: String, bookId: String) = "/api/android/v1/reading/${segment(libraryId)}/${segment(bookId)}"
}

internal fun PdfServerPosition.pageFor(pageCount: Int): Int {
    val last = (pageCount - 1).coerceAtLeast(0)
    return page?.coerceIn(0, last) ?: kotlin.math.round(fraction.coerceIn(0.0, 1.0) * last).toInt()
}
