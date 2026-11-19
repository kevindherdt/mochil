package net.kdhmedia.mochil.video

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.playback.shouldUploadPassiveProgress
import net.kdhmedia.mochil.playback.readPlaybackProgressResponse
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.reportIfExpiredDeviceSession
import org.json.JSONObject

/** Revision-aware playback progress; passive players never replace newer remote state. */
internal class VideoProgressClient {
    private val known = object : LinkedHashMap<String, VideoProgressSnapshot>(32, .75f, true) {
        override fun removeEldestEntry(eldest: MutableMap.MutableEntry<String, VideoProgressSnapshot>?): Boolean = size > MAX_KNOWN_ITEMS
    }

    @Synchronized
    fun save(credential: DeviceCredential, entry: VideoPlaybackEntry, positionSeconds: Double, durationSeconds: Double, finished: Boolean, explicitSeek: Boolean = false, initialRevision: Long = entry.progressRevision, onSaved: (Long, Long) -> Unit = { _, _ -> }, maySend: () -> Boolean = { true }): Boolean {
        if (!maySend()) return false
        if (entry.libraryId.isBlank() || entry.itemId.isBlank() || !positionSeconds.isFinite() || !durationSeconds.isFinite() || durationSeconds <= 0.0) return false
        if (untouchedFinishedReplay(entry, positionSeconds, finished, explicitSeek)) return true
        val key = videoProgressCacheKey(credential, entry.libraryId, entry.itemId)
        val path = "/api/android/v1/video-progress/${encode(entry.libraryId)}/${encode(entry.itemId)}"
        // A rebased revision identifies our last successful replay save, not its position.
        // Fetch the actual state before comparing passive progress so we cannot rewind it.
        var remote = known[key] ?: if (entry.finished && initialRevision != entry.progressRevision) {
            val current = get(credential, path) ?: return false
            if (!maySend()) return false
            current.copy(replayOwnerRevision = if (current.revision == initialRevision) entry.progressRevision else null).also { known[key] = it }
        } else {
            VideoProgressSnapshot(initialRevision, entry.positionMs / 1_000.0, entry.finished).also { known[key] = it }
        }
        if (!explicitSeek && entry.finished && remote.rejectedReplayRevision == entry.progressRevision) return true
        if (!explicitSeek && entry.finished && remote.revision != entry.progressRevision && remote.replayOwnerRevision != entry.progressRevision) return true
        // Replaying a known finished snapshot is a conditional restart, not a seek
        // that may be retried against a newer revision from another device.
        val restart = entry.finished && remote.finished && remote.revision == entry.progressRevision && !finished
        if (!restart && !shouldUploadVideoProgress(remote.positionSeconds, remote.finished, positionSeconds, durationSeconds, finished, explicitSeek)) return true
        when (val result = put(credential, path, positionSeconds, durationSeconds, finished, remote.revision, explicitSeek || restart, entry.revision)) {
            is VideoProgressSaveResult.Saved -> {
                known[key] = result.progress.copy(replayOwnerRevision = if (entry.finished) entry.progressRevision else null)
                onSaved(remote.revision, result.progress.revision)
            }
            VideoProgressSaveResult.Conflict -> {
                if (!maySend()) return false
                remote = get(credential, path)?.also { known[key] = it } ?: return false
                if (entry.finished && !explicitSeek) {
                    known[key] = remote.copy(rejectedReplayRevision = entry.progressRevision)
                    return maySend()
                }
                if (!maySend()) return false
                if (!shouldUploadVideoProgress(remote.positionSeconds, remote.finished, positionSeconds, durationSeconds, finished, explicitSeek)) return true
                when (val retry = put(credential, path, positionSeconds, durationSeconds, finished, remote.revision, explicitSeek, entry.revision)) {
                    is VideoProgressSaveResult.Saved -> {
                        known[key] = retry.progress.copy(replayOwnerRevision = if (entry.finished) entry.progressRevision else null)
                        onSaved(remote.revision, retry.progress.revision)
                    }
                    VideoProgressSaveResult.Conflict -> { known.remove(key); return false }
                }
            }
        }
        return true
    }

    private fun get(credential: DeviceCredential, path: String): VideoProgressSnapshot? {
        val connection = openAndroidApiGet(credential, path, readTimeoutMillis = 15_000)
        return try {
            if (connection.responseCode !in 200..299) return null
            connection.inputStream.use { it.readPlaybackProgressResponse() }.snapshot()
        } finally { connection.disconnect() }
    }

    private fun put(credential: DeviceCredential, path: String, position: Double, duration: Double, finished: Boolean, revision: Long, explicitSeek: Boolean, mediaRevision: String): VideoProgressSaveResult {
        val connection = openMutation(credential, path).apply {
            requestMethod = "PUT"
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        val body = videoProgressRequestBody(
            positionSeconds = position,
            durationSeconds = duration,
            finished = finished,
            revision = revision,
            explicitSeek = explicitSeek,
            deviceId = credential.deviceId,
            mediaRevision = mediaRevision,
        )
        return try {
            connection.outputStream.use { it.write(body.toByteArray()) }
            val status = connection.responseCode
            reportIfExpiredDeviceSession(status)
            when (status) {
                in 200..299 -> VideoProgressSaveResult.Saved(connection.inputStream.use { it.readPlaybackProgressResponse() }.snapshot())
                409 -> VideoProgressSaveResult.Conflict
                else -> error("Video progress update failed.")
            }
        } finally { connection.disconnect() }
    }

    private fun openMutation(credential: DeviceCredential, path: String) =
        (URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 10_000
            readTimeout = 15_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    private fun JSONObject.snapshot() = VideoProgressSnapshot(
        optLong("revision"), optDouble("position_seconds").takeIf(Double::isFinite) ?: 0.0, optBoolean("finished"),
    )

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")
    private companion object { const val MAX_KNOWN_ITEMS = 256 }
}

/** Video progress revisions belong to the paired account represented by this device credential. */
internal fun videoProgressCacheKey(credential: DeviceCredential, libraryId: String, itemId: String): String =
    "${credential.serverUrl}\u0000${credential.deviceId}\u0000$libraryId\u0000$itemId"

internal fun untouchedFinishedReplay(entry: VideoPlaybackEntry, position: Double, finished: Boolean, explicitSeek: Boolean): Boolean =
    entry.finished && position <= 0.0 && !finished && !explicitSeek

private data class VideoProgressSnapshot(val revision: Long, val positionSeconds: Double, val finished: Boolean, val rejectedReplayRevision: Long? = null, val replayOwnerRevision: Long? = null)
private sealed interface VideoProgressSaveResult {
    data class Saved(val progress: VideoProgressSnapshot) : VideoProgressSaveResult
    data object Conflict : VideoProgressSaveResult
}

internal fun shouldUploadVideoProgress(remotePosition: Double, remoteFinished: Boolean, localPosition: Double, duration: Double, localFinished: Boolean, explicitSeek: Boolean = false): Boolean {
    return (explicitSeek && localPosition.isFinite() && duration.isFinite() && duration > 0.0) ||
        shouldUploadPassiveProgress(remotePosition, remoteFinished, localPosition, duration, localFinished)
}

internal fun videoProgressRequestBody(
    positionSeconds: Double,
    durationSeconds: Double,
    finished: Boolean,
    revision: Long,
    explicitSeek: Boolean,
    deviceId: String,
    mediaRevision: String = "",
): String = JSONObject()
    .put("position_seconds", positionSeconds.coerceIn(0.0, durationSeconds))
    .put("duration_seconds", durationSeconds)
    .put("finished", finished)
    .put("expected_revision", revision)
    .put("explicit_seek", explicitSeek)
    .put("device_id", deviceId.take(256))
    .apply { if (mediaRevision.isNotBlank()) put("expected_media_revision", mediaRevision) }
    .toString()
