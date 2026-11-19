package net.kdhmedia.mochil.playback

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.reportIfExpiredDeviceSession
import org.json.JSONObject

class AudioProgressClient {
    private val known = object : LinkedHashMap<String, RemoteAudioProgress>(32, .75f, true) {
        override fun removeEldestEntry(eldest: MutableMap.MutableEntry<String, RemoteAudioProgress>?): Boolean = size > MAX_KNOWN_ITEMS
    }

    @Synchronized
    fun save(credential: DeviceCredential, libraryId: String, itemId: String, positionSeconds: Double, durationSeconds: Double, explicitSeek: Boolean = false, maySend: () -> Boolean = { true }): Boolean {
        if (!maySend()) return false
        if (libraryId.isBlank() || itemId.isBlank() || durationSeconds <= 0) return true
        val path = "/api/android/v1/audio-progress/${encode(libraryId)}/${encode(itemId)}"
        val cacheKey = audioProgressCacheKey(credential, libraryId, itemId)
        var remote = known[cacheKey] ?: getProgress(credential, path)?.also { known[cacheKey] = it } ?: return false
        if (!maySend()) return false
        if (!shouldUploadAudioProgress(remote.positionSeconds, remote.finished, positionSeconds, durationSeconds, explicitSeek)) return true
        when (val result = putProgress(credential, path, positionSeconds, durationSeconds, remote.revision, explicitSeek)) {
            is AudioProgressSaveResult.Saved -> known[cacheKey] = result.progress
            AudioProgressSaveResult.Conflict -> {
                if (!maySend()) return false
                remote = getProgress(credential, path)?.also { known[cacheKey] = it } ?: return false
                if (!maySend()) return false
                if (!shouldUploadAudioProgress(remote.positionSeconds, remote.finished, positionSeconds, durationSeconds, explicitSeek)) return true
                when (val retry = putProgress(credential, path, positionSeconds, durationSeconds, remote.revision, explicitSeek)) {
                    is AudioProgressSaveResult.Saved -> known[cacheKey] = retry.progress
                    AudioProgressSaveResult.Conflict -> { known.remove(cacheKey); return false }
                }
            }
        }
        return true
    }

    private fun putProgress(credential: DeviceCredential, path: String, positionSeconds: Double, durationSeconds: Double, revision: Long, explicitSeek: Boolean): AudioProgressSaveResult {
        val connection = openMutation(credential, path).apply {
            requestMethod = "PUT"
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        val finished = audioProgressFinished(positionSeconds, durationSeconds)
        val body = audioProgressRequestBody(
            positionSeconds = positionSeconds,
            durationSeconds = durationSeconds,
            revision = revision,
            explicitSeek = explicitSeek,
            deviceId = credential.deviceId,
        )
        return try {
            connection.outputStream.use { it.write(body.toByteArray()) }
            val status = connection.responseCode
            reportIfExpiredDeviceSession(status)
            when (status) {
                in 200..299 -> {
                    val value = connection.inputStream.use { it.readPlaybackProgressResponse() }
                    AudioProgressSaveResult.Saved(RemoteAudioProgress(value.optLong("revision", revision + 1), value.optDouble("position_seconds", positionSeconds), value.optBoolean("finished", finished)))
                }
                409 -> AudioProgressSaveResult.Conflict
                else -> error("Progress update failed.")
            }
        } finally { connection.disconnect() }
    }

    private fun getProgress(credential: DeviceCredential, path: String): RemoteAudioProgress? {
        val connection = openAndroidApiGet(credential, path)
        return try {
            when (connection.responseCode) {
                in 200..299 -> connection.inputStream.use { it.readPlaybackProgressResponse() }.run {
                    RemoteAudioProgress(optLong("revision"), optDouble("position_seconds"), optBoolean("finished"))
                }
                404 -> RemoteAudioProgress(0, 0.0, false)
                else -> null
            }
        } finally { connection.disconnect() }
    }

    private fun openMutation(credential: DeviceCredential, path: String) =
        (URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 10_000
            readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")

    private companion object { const val MAX_KNOWN_ITEMS = 256 }
}

/** Progress revisions are account-scoped; a newly paired device must never reuse another credential's cache. */
internal fun audioProgressCacheKey(credential: DeviceCredential, libraryId: String, itemId: String): String =
    "${credential.serverUrl}\u0000${credential.deviceId}\u0000$libraryId\u0000$itemId"

private data class RemoteAudioProgress(val revision: Long, val positionSeconds: Double, val finished: Boolean)
private sealed interface AudioProgressSaveResult {
    data class Saved(val progress: RemoteAudioProgress) : AudioProgressSaveResult
    data object Conflict : AudioProgressSaveResult
}

internal fun shouldUploadAudioProgress(remotePosition: Double, remoteFinished: Boolean, localPosition: Double, duration: Double, explicitSeek: Boolean = false): Boolean {
    return (explicitSeek && localPosition.isFinite() && duration.isFinite() && duration > 0.0) ||
        shouldUploadPassiveProgress(
            remotePosition,
            remoteFinished,
            localPosition,
            duration,
            localFinished = audioProgressFinished(localPosition, duration),
        )
}

internal fun audioProgressRequestBody(
    positionSeconds: Double,
    durationSeconds: Double,
    revision: Long,
    explicitSeek: Boolean,
    deviceId: String,
): String = JSONObject().apply {
    val boundedPosition = positionSeconds.coerceIn(0.0, durationSeconds)
    put("position_seconds", boundedPosition)
    put("duration_seconds", durationSeconds)
    put("finished", audioProgressFinished(boundedPosition, durationSeconds))
    put("playback_rate", 1.0)
    put("expected_revision", revision)
    put("explicit_seek", explicitSeek)
    put("device_id", deviceId.take(256))
}.toString()
