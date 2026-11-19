package net.kdhmedia.mochil.reader.epub

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.reader.currentReaderDeviceName
import org.json.JSONObject
import org.readium.r2.shared.publication.Locator

data class ServerTextAnchor(
    val prefix: String,
    val exact: String,
    val suffix: String,
    val spineIndex: Int,
    val progress: Double,
)

data class ServerReadingPosition(
    val revision: Long,
    val updatedAt: Long,
    val fraction: Double,
    val locators: Map<String, String>,
    val koReaderLocator: String,
    val anchor: ServerTextAnchor?,
    val deviceId: String,
    val deviceName: String,
)

sealed interface ReadingSaveResult {
    data class Saved(val position: ServerReadingPosition) : ReadingSaveResult
    data class Conflict(val position: ServerReadingPosition) : ReadingSaveResult
}

internal interface ReadingProgressGateway {
    suspend fun get(credential: DeviceCredential, libraryId: String, bookId: String): ServerReadingPosition?
    suspend fun put(credential: DeviceCredential, libraryId: String, bookId: String, locator: Locator, fraction: Double, baseRevision: Long, spineIndex: Int, koReaderLocator: String?): ReadingSaveResult
    suspend fun getExternal(credential: DeviceCredential, titleId: String): ServerReadingPosition?
    suspend fun putExternal(credential: DeviceCredential, titleId: String, locator: Locator, fraction: Double, baseRevision: Long): ReadingSaveResult
}

class ReadingProgressClient : ReadingProgressGateway {
    override suspend fun get(credential: DeviceCredential, libraryId: String, bookId: String): ServerReadingPosition? = withContext(Dispatchers.IO) {
        val connection = openAndroidApiGet(credential, path(libraryId, bookId))
        try {
            if (connection.responseCode == 404) return@withContext null
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode} while loading reading progress.")
            decode(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }))
        } finally { connection.disconnect() }
    }

    override suspend fun put(
        credential: DeviceCredential,
        libraryId: String,
        bookId: String,
        locator: Locator,
        fraction: Double,
        baseRevision: Long,
        spineIndex: Int,
        koReaderLocator: String?,
    ): ReadingSaveResult = withContext(Dispatchers.IO) {
        val locatorJSON = locator.boundedLocatorJSON()
        val body = JSONObject().apply {
            put("fraction", fraction.coerceIn(0.0, 1.0))
            put("base_revision", baseRevision)
            put("device_id", credential.deviceId)
            put("device_name", currentReaderDeviceName())
            put("locators", JSONObject().put("readium", locatorJSON))
            put("web_anchor", locator.anchorJSON(spineIndex))
            koReaderLocator?.takeIf { KoReaderXPointer.parse(it) != null }?.let { put("koreader_locator", it) }
        }.toString().encodeToByteArray()
        val connection = mutationConnection(credential, path(libraryId, bookId), "PUT").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setFixedLengthStreamingMode(body.size)
        }
        try {
            connection.outputStream.use { it.write(body) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299 && status != 409) error("The server returned $status while saving reading progress.")
            val stream = if (status == 409) connection.errorStream ?: connection.inputStream else connection.inputStream
            val position = decode(JSONObject(stream.use { it.readBoundedResponseText() }))
            if (status == 409) ReadingSaveResult.Conflict(position) else ReadingSaveResult.Saved(position)
        } finally { connection.disconnect() }
    }

    override suspend fun getExternal(credential: DeviceCredential, titleId: String): ServerReadingPosition? = withContext(Dispatchers.IO) {
        val connection = openAndroidApiGet(credential, externalPath(titleId) + "?chapter=publication")
        try {
            if (connection.responseCode == 404) return@withContext null
            if (connection.responseCode !in 200..299) error("The server returned ${connection.responseCode} while loading external reading progress.")
            decodeExternal(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }))
        } finally { connection.disconnect() }
    }

    override suspend fun putExternal(
        credential: DeviceCredential,
        titleId: String,
        locator: Locator,
        fraction: Double,
        baseRevision: Long,
    ): ReadingSaveResult = withContext(Dispatchers.IO) {
        val body = JSONObject().apply {
            put("base_revision", baseRevision)
            put("progress", JSONObject().apply {
                put("remote_chapter_id", "publication")
                put("locator", locator.boundedLocatorJSON())
                put("fraction", fraction.coerceIn(0.0, 1.0))
                put("device_id", credential.deviceId)
                put("device_name", currentReaderDeviceName())
                put("finished", fraction >= 0.995)
            })
        }.toString().encodeToByteArray()
        val connection = mutationConnection(credential, externalPath(titleId), "PUT").apply {
            doOutput = true; setRequestProperty("Content-Type", "application/json"); setFixedLengthStreamingMode(body.size)
        }
        try {
            connection.outputStream.use { it.write(body) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299 && status != 409) error("The server returned $status while saving external reading progress.")
            val stream = if (status == 409) connection.errorStream ?: connection.inputStream else connection.inputStream
            val position = decodeExternal(JSONObject(stream.use { it.readBoundedResponseText() }))
            if (status == 409) ReadingSaveResult.Conflict(position) else ReadingSaveResult.Saved(position)
        } finally { connection.disconnect() }
    }

    private fun mutationConnection(credential: DeviceCredential, path: String, method: String): HttpURLConnection =
        URI(credential.serverUrl + path).toURL().openConnection().let { it as HttpURLConnection }.apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    private fun path(libraryId: String, bookId: String) = "/api/android/v1/reading/${encode(libraryId)}/${encode(bookId)}"
    private fun externalPath(titleId: String) = "/api/android/v1/external-library/${encode(titleId)}/progress"
    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")

    private fun decode(json: JSONObject): ServerReadingPosition {
        val values = json.optJSONObject("locators")
        val locators = buildMap {
            if (values != null) values.keys().forEach { key -> values.optString(key).takeIf(String::isNotBlank)?.let { put(key, it) } }
        }
        val anchor = json.optJSONObject("web_anchor")?.let {
            ServerTextAnchor(it.optString("prefix"), it.optString("exact"), it.optString("suffix"), it.optInt("spine_index", -1), it.optDouble("progress", 0.0))
        }?.takeIf { it.spineIndex >= 0 }
        return ServerReadingPosition(
            json.optLong("revision"), json.optLong("updated_at"), json.optDouble("fraction"), locators,
            json.optString("koreader_locator"), anchor, json.optString("device_id"), json.optString("device_name"),
        )
    }

    private fun decodeExternal(json: JSONObject): ServerReadingPosition = ServerReadingPosition(
        revision = json.optLong("revision"), updatedAt = json.optLong("updated_at"), fraction = json.optDouble("fraction"),
        locators = json.optString("locator").takeIf(String::isNotBlank)?.let { mapOf("readium" to it) }.orEmpty(),
        koReaderLocator = "", anchor = null, deviceId = json.optString("device_id"), deviceName = json.optString("device_name"),
    )
}

private fun Locator.anchorJSON(spineIndex: Int): JSONObject = JSONObject().apply {
    // A plain page position has no selected text, so the quote is empty and other
    // readers cannot find the page. The text just after the position stands in.
    val quote = text.highlight.orEmpty().ifBlank { text.after.orEmpty().take(160) }
    val following = if (text.highlight.isNullOrBlank()) "" else text.after.orEmpty()
    put("version", 1)
    put("prefix", text.before.orEmpty().takeLast(512))
    put("exact", quote.take(1024))
    put("suffix", following.take(512))
    put("text", quote.take(1024))
    put("spine_index", spineIndex.coerceAtLeast(0))
    put("progress", (locations.progression ?: 0.0).coerceIn(0.0, 1.0))
}

private fun Locator.boundedLocatorJSON(): String {
    val withText = copy(
        text = Locator.Text(
            text.before.boundedUtf8FromEnd(512),
            text.highlight.boundedUtf8(1024),
            text.after.boundedUtf8(512),
        ),
    ).toJSON().toString()
    if (withText.encodeToByteArray().size <= MAXIMUM_LOCATOR_BYTES) return withText

    // Some EPUBs generate exceptionally deep CSS selectors. A locator is useful
    // for precision, but it must never prevent the durable fraction, anchor, and
    // KoReader-compatible location from being saved. Preserve only the stable
    // resource and total progression when the renderer's full locator is large.
    val source = JSONObject(withText)
    val compact = JSONObject().apply {
        put("href", source.optString("href"))
        source.optString("type").takeIf(String::isNotBlank)?.let { put("type", it) }
        source.optJSONObject("locations")
            ?.optDouble("totalProgression", Double.NaN)
            ?.takeIf(Double::isFinite)
            ?.let { put("locations", JSONObject().put("totalProgression", it.coerceIn(0.0, 1.0))) }
    }.toString()
    require(compact.encodeToByteArray().size <= MAXIMUM_LOCATOR_BYTES) { "The EPUB locator is too large to synchronize." }
    return compact
}

private const val MAXIMUM_LOCATOR_BYTES = 16_384

private fun String?.boundedUtf8(maximumBytes: Int): String? {
    val value = this ?: return null
    if (value.encodeToByteArray().size <= maximumBytes) return value
    val result = StringBuilder()
    var index = 0
    var bytes = 0
    while (index < value.length) {
        val characters = Character.toChars(value.codePointAt(index))
        val encodedBytes = String(characters).encodeToByteArray().size
        if (bytes + encodedBytes > maximumBytes) break
        result.append(characters)
        bytes += encodedBytes
        index += characters.size
    }
    return result.toString()
}

private fun String?.boundedUtf8FromEnd(maximumBytes: Int): String? {
    val value = this ?: return null
    if (value.encodeToByteArray().size <= maximumBytes) return value
    val result = StringBuilder()
    var index = value.length
    var bytes = 0
    while (index > 0) {
        val codePoint = value.codePointBefore(index)
        val characters = Character.toChars(codePoint)
        val encodedBytes = String(characters).encodeToByteArray().size
        if (bytes + encodedBytes > maximumBytes) break
        result.insert(0, characters)
        bytes += encodedBytes
        index -= characters.size
    }
    return result.toString()
}
