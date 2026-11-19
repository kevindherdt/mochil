package net.kdhmedia.mochil.video

import java.security.MessageDigest
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONArray
import org.json.JSONObject

internal const val MAX_PENDING_VIDEO_PROGRESS = 64
internal const val MAX_VIDEO_PROGRESS_JOURNAL_BYTES = 256 * 1024

/** No stream URLs, tokens, titles or subtitle data belong in a progress checkpoint. */
internal data class PendingVideoProgress(
    val owner: String,
    val libraryId: String,
    val itemId: String,
    val positionSeconds: Double,
    val durationSeconds: Double,
    val revision: Long,
    val finished: Boolean,
    val explicitSeekAtEpochMs: Long = 0,
    val mediaRevision: String = "",
    val replayFinished: Boolean = false,
    val rebasedFromRevision: Long? = null,
) {
    fun valid(): Boolean = owner.matches(VIDEO_PROGRESS_OWNER) &&
        libraryId.isNotBlank() && libraryId.length <= 160 &&
        itemId.isNotBlank() && itemId.length <= 512 &&
        positionSeconds.isFinite() && durationSeconds.isFinite() && durationSeconds > 0 &&
        positionSeconds in 0.0..durationSeconds && revision >= 0 && explicitSeekAtEpochMs >= 0 && mediaRevision.length <= 160

    fun explicitSeekForReplay(nowEpochMs: Long): Boolean = explicitSeekAtEpochMs > 0 &&
        nowEpochMs >= explicitSeekAtEpochMs && nowEpochMs - explicitSeekAtEpochMs <= 600_000
}

internal fun videoProgressOwner(credential: DeviceCredential): String = MessageDigest.getInstance("SHA-256")
    .digest(JSONArray().put(credential.serverUrl).put(credential.deviceId).toString().encodeToByteArray())
    .joinToString("") { "%02x".format(it) }

internal fun mergePendingVideoProgress(current: List<PendingVideoProgress>, value: PendingVideoProgress): List<PendingVideoProgress> {
    require(value.valid())
    val same: (PendingVideoProgress) -> Boolean = { it.owner == value.owner && it.libraryId == value.libraryId && it.itemId == value.itemId }
    val prior = current.lastOrNull { same(it) && it.mediaRevision == value.mediaRevision }
    val rebased = if (prior != null && prior.rebasedFromRevision == value.revision &&
        prior.replayFinished == value.replayFinished) value.copy(revision = prior.revision, rebasedFromRevision = prior.rebasedFromRevision) else value
    val merged = if (rebased.explicitSeekAtEpochMs == 0L && prior != null) {
        rebased.copy(explicitSeekAtEpochMs = prior.explicitSeekAtEpochMs)
    } else rebased
    return (current.filter { it.valid() && !same(it) } + merged).takeLast(MAX_PENDING_VIDEO_PROGRESS)
}

internal fun encodePendingVideoProgress(values: List<PendingVideoProgress>): ByteArray {
    require(values.size <= MAX_PENDING_VIDEO_PROGRESS && values.all { it.valid() })
    val encoded = JSONArray().apply {
        values.forEach { value -> put(JSONObject()
            .put("owner", value.owner).put("library", value.libraryId).put("item", value.itemId)
            .put("position", value.positionSeconds).put("duration", value.durationSeconds)
            .put("revision", value.revision).put("finished", value.finished)
            .put("seek_at", value.explicitSeekAtEpochMs).put("media_revision", value.mediaRevision).put("replay_finished", value.replayFinished)
            .apply { value.rebasedFromRevision?.let { put("rebased_from_revision", it) } }) }
    }.toString().encodeToByteArray()
    require(encoded.size <= MAX_VIDEO_PROGRESS_JOURNAL_BYTES)
    return encoded
}

/** Malformed data throws, so callers can tell it from a valid empty journal. */
internal fun decodePendingVideoProgressStrict(bytes: ByteArray): List<PendingVideoProgress> {
    require(bytes.size <= MAX_VIDEO_PROGRESS_JOURNAL_BYTES)
    val values = JSONArray(bytes.decodeToString())
    require(values.length() <= MAX_PENDING_VIDEO_PROGRESS)
    return List(values.length()) { index -> values.getJSONObject(index).let { value ->
        PendingVideoProgress(value.getString("owner"), value.getString("library"), value.getString("item"),
            value.getDouble("position"), value.getDouble("duration"), value.getLong("revision"),
            value.getBoolean("finished"), value.optLong("seek_at", 0), value.optString("media_revision"), value.optBoolean("replay_finished"),
            if (value.has("rebased_from_revision")) value.getLong("rebased_from_revision") else null).also { require(it.valid()) }
    } }
}

private val VIDEO_PROGRESS_OWNER = Regex("[0-9a-f]{64}")
