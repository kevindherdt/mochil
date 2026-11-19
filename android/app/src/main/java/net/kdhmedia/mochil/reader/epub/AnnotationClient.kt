package net.kdhmedia.mochil.reader.epub

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import net.kdhmedia.mochil.network.openAndroidApiGet
import org.json.JSONArray
import org.json.JSONObject
import org.readium.r2.shared.publication.Locator

data class SyncedAnnotation(
    val id: String,
    val kind: String,
    val quote: String = "",
    val note: String = "",
    val color: String = "",
    val chapter: String = "",
    val createdAt: Long = 0,
    val updatedAt: Long = 0,
    val revision: Long = 0,
    val deleted: Boolean = false,
    val deviceId: String = "",
    val locators: Map<String, String> = emptyMap(),
    val anchor: ServerTextAnchor? = null,
)

data class AnnotationChange(val baseRevision: Long, val annotation: SyncedAnnotation)
data class AnnotationConflict(val submitted: SyncedAnnotation, val current: SyncedAnnotation)
data class AnnotationSyncResult(val revision: Long, val applied: List<SyncedAnnotation>, val conflicts: List<AnnotationConflict>)

class AnnotationClient {
    suspend fun changes(credential: DeviceCredential, libraryId: String, bookId: String, since: Long): AnnotationSyncResult =
        request(credential, libraryId, bookId, "GET", since = since)

    suspend fun apply(credential: DeviceCredential, libraryId: String, bookId: String, changes: List<AnnotationChange>): AnnotationSyncResult {
        require(changes.isNotEmpty() && changes.size <= MAX_BATCH_SIZE)
        val values = JSONArray()
        changes.forEach { change ->
            values.put(JSONObject().apply {
                put("base_revision", change.baseRevision)
                put("annotation", encodeSyncedAnnotation(change.annotation))
            })
        }
        return request(credential, libraryId, bookId, "PUT", JSONObject().put("changes", values).toString().encodeToByteArray())
    }

    private suspend fun request(
        credential: DeviceCredential,
        libraryId: String,
        bookId: String,
        method: String,
        body: ByteArray? = null,
        since: Long? = null,
    ): AnnotationSyncResult = withContext(Dispatchers.IO) {
        val suffix = since?.let { "?since=${it.coerceAtLeast(0)}" }.orEmpty()
        val requestPath = path(libraryId, bookId) + suffix
        val connection = if (body == null && method == "GET") {
            openAndroidApiGet(credential, requestPath)
        } else {
            val payload = requireNotNull(body)
            URI(credential.serverUrl + requestPath).toURL().openConnection().let { it as HttpURLConnection }.apply {
                requestMethod = method
                connectTimeout = 10_000
                readTimeout = 20_000
                instanceFollowRedirects = false
                setRequestProperty("Authorization", "Bearer ${credential.token}")
                setRequestProperty("Accept", "application/json")
                require(payload.size <= MAX_BODY_BYTES)
                doOutput = true
                setRequestProperty("Content-Type", "application/json")
                setFixedLengthStreamingMode(payload.size)
            }
        }
        try {
            if (body != null) connection.outputStream.use { it.write(body) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status while synchronizing annotations.")
            decode(JSONObject(connection.inputStream.use { it.readBoundedResponseText() }))
        } finally {
            connection.disconnect()
        }
    }

    private fun path(libraryId: String, bookId: String) = "/api/android/v1/annotations/${encode(libraryId)}/${encode(bookId)}"
    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")

    private fun decode(json: JSONObject): AnnotationSyncResult = AnnotationSyncResult(
        revision = json.optLong("revision"),
        applied = json.optJSONArray("applied").toAnnotations(),
        conflicts = json.optJSONArray("conflicts").let { values ->
            if (values == null) emptyList() else List(values.requireAtMost(MAX_BATCH_SIZE, "Annotation conflicts").length()) { index ->
                values.getJSONObject(index).run { AnnotationConflict(decodeSyncedAnnotation(getJSONObject("submitted")), decodeSyncedAnnotation(getJSONObject("current"))) }
            }
        },
    )

    private companion object {
        const val MAX_BATCH_SIZE = 500
        const val MAX_BODY_BYTES = 4 * 1024 * 1024
    }
}

internal fun encodeSyncedAnnotation(value: SyncedAnnotation) = JSONObject().apply {
    with(value) {
        put("id", id.take(128))
        put("kind", kind)
        put("quote", quote.take(8192))
        put("note", note.take(16384))
        put("color", color.take(32))
        put("chapter", chapter.take(1024))
        put("created_at", createdAt)
        put("updated_at", updatedAt)
        put("revision", revision)
        put("deleted", deleted)
        put("device_id", deviceId.take(128))
        put("locators", JSONObject(locators))
        anchor?.let { value ->
            put("anchor", JSONObject().apply {
                put("version", 1)
                put("prefix", value.prefix.takeLast(512))
                put("exact", value.exact.take(1024))
                put("suffix", value.suffix.take(512))
                put("spine_index", value.spineIndex.coerceAtLeast(0))
                put("progress", value.progress.coerceIn(0.0, 1.0))
            })
        }
    }
}

private fun JSONArray?.toAnnotations(): List<SyncedAnnotation> =
    if (this == null) emptyList() else List(requireAtMost(500, "Annotations").length()) { decodeSyncedAnnotation(getJSONObject(it)) }

internal fun decodeSyncedAnnotation(json: JSONObject): SyncedAnnotation = with(json) {
    val encodedLocators = optJSONObject("locators")
    val locators = buildMap {
        encodedLocators?.keys()?.forEach { key -> encodedLocators.optString(key).takeIf(String::isNotBlank)?.let { put(key, it) } }
    }
    val anchor = optJSONObject("anchor")?.let {
        ServerTextAnchor(it.optString("prefix"), it.optString("exact"), it.optString("suffix"), it.optInt("spine_index", -1), it.optDouble("progress", 0.0))
    }?.takeIf { it.spineIndex >= 0 }
    SyncedAnnotation(
        id = optString("id"), kind = optString("kind"), quote = optString("quote"), note = optString("note"), color = optString("color"),
        chapter = optString("chapter"), createdAt = optLong("created_at"), updatedAt = optLong("updated_at"), revision = optLong("revision"),
        deleted = optBoolean("deleted"), deviceId = optString("device_id"), locators = locators, anchor = anchor,
    )
}

fun SyncedAnnotation.readiumLocator(): Locator? = locators["readium"]?.let { encoded ->
    runCatching { Locator.fromJSON(JSONObject(encoded)) }.getOrNull()
}
