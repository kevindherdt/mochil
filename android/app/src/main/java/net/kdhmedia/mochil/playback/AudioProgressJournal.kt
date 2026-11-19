package net.kdhmedia.mochil.playback

import android.content.Context
import java.security.MessageDigest
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import org.json.JSONArray
import org.json.JSONObject

internal data class PendingAudioProgress(
    val libraryId: String,
    val itemId: String,
    val positionSeconds: Double,
    val durationSeconds: Double,
    val explicitSeek: Boolean = false,
    val explicitSeekAtEpochMs: Long = 0,
    val owner: String = "",
)

/** Small encrypted outbox for final positions that must survive task removal. */
internal class AudioProgressJournal(
    private val load: () -> ByteArray?,
    private val persist: (ByteArray) -> Boolean,
    private val clear: () -> Boolean,
) {
    private constructor(storage: EncryptedPayloadStore) : this(storage::loadStrict, storage::save, storage::clear)
    constructor(context: Context) : this(EncryptedPayloadStore(context.applicationContext,
        "audio_progress_outbox", "mochil_audio_progress_outbox"))

    fun entries(deviceId: String): List<PendingAudioProgress> = synchronized(journalLock) {
        val current = decode() ?: return@synchronized emptyList()
        readAudioProgressForOwner(current, deviceId, ::encode)
    }

    fun put(value: PendingAudioProgress): PendingAudioProgress? = synchronized(journalLock) {
        if (!value.owner.matches(AUDIO_PROGRESS_OWNER)) return@synchronized null
        val current = decode() ?: return@synchronized null
        commitPendingAudioProgress(current, value, MAX_ENTRIES, ::encode)
    }

    fun removeIfCurrent(value: PendingAudioProgress): Unit = synchronized(journalLock) {
        val current = decode() ?: return@synchronized
        val next = current.filterNot { it == value }
        if (next.size != current.size) encode(next)
    }

    private fun decode(): List<PendingAudioProgress>? = runCatching {
        val values = JSONArray(load()?.decodeToString() ?: "[]")
        List(minOf(values.length(), MAX_ENTRIES)) { index ->
            values.getJSONObject(index).run {
                PendingAudioProgress(
                    getString("library"),
                    getString("item"),
                    getDouble("position"),
                    getDouble("duration"),
                    optBoolean("explicit_seek", false),
                    optLong("explicit_seek_at_ms", 0),
                    optString("owner"),
                )
            }
        }.filter { it.libraryId.isNotBlank() && it.itemId.isNotBlank() && it.positionSeconds.isFinite() && it.durationSeconds.isFinite() && it.durationSeconds > 0 }
    }.getOrNull()

    private fun encode(values: List<PendingAudioProgress>): Boolean {
        if (values.isEmpty()) {
            return clear()
        }
        val data = JSONArray().apply { values.forEach { value -> put(JSONObject().apply {
            put("library", value.libraryId); put("item", value.itemId)
            put("position", value.positionSeconds); put("duration", value.durationSeconds)
            put("owner", value.owner)
            if (value.explicitSeek) {
                put("explicit_seek", true)
                put("explicit_seek_at_ms", value.explicitSeekAtEpochMs)
            }
        }) } }.toString().encodeToByteArray()
        return persist(data)
    }

    private companion object {
        const val MAX_ENTRIES = 64
        // Activity/service recreation can overlap instances sharing one encrypted outbox.
        val journalLock = Any()
    }
}

internal fun mergePendingAudioProgress(current: List<PendingAudioProgress>, value: PendingAudioProgress, limit: Int): List<PendingAudioProgress> {
    val key = value.owner + "\u0000" + value.libraryId + "\u0000" + value.itemId
    val retained = current.firstOrNull { it.owner + "\u0000" + it.libraryId + "\u0000" + it.itemId == key }?.takeIf { it.explicitSeek }
    val merged = if (retained != null && !value.explicitSeek) {
        value.copy(explicitSeek = true, explicitSeekAtEpochMs = retained.explicitSeekAtEpochMs)
    } else value
    return (current.filterNot { it.owner + "\u0000" + it.libraryId + "\u0000" + it.itemId == key } + merged).takeLast(limit.coerceAtLeast(0))
}

internal fun audioProgressEntriesForOwner(values: List<PendingAudioProgress>, deviceId: String): List<PendingAudioProgress> {
    val owner = audioProgressOwnerIdentity(deviceId)
    return values.filter { it.owner == owner }
}

/** Reads do not encrypt/commit again unless account filtering actually prunes rows. */
internal fun readAudioProgressForOwner(
    current: List<PendingAudioProgress>, deviceId: String,
    persist: (List<PendingAudioProgress>) -> Boolean,
): List<PendingAudioProgress> {
    val retained = audioProgressEntriesForOwner(current, deviceId)
    if (retained.size != current.size) persist(retained)
    return retained
}

internal fun audioProgressOwnerIdentity(deviceId: String): String = MessageDigest.getInstance("SHA-256")
    .digest(deviceId.trim().encodeToByteArray()).take(12).joinToString("") { "%02x".format(it) }

private val AUDIO_PROGRESS_OWNER = Regex("[0-9a-f]{24}")

internal const val EXPLICIT_SEEK_REPLAY_WINDOW_MS = 10 * 60 * 1_000L

/** Explicit backward ownership is user intent, but it must not survive as a permanent stale-write override. */
internal fun PendingAudioProgress.explicitSeekForReplay(nowEpochMs: Long): Boolean {
    if (!explicitSeek || explicitSeekAtEpochMs <= 0) return false
    val age = nowEpochMs - explicitSeekAtEpochMs
    return age in 0..EXPLICIT_SEEK_REPLAY_WINDOW_MS
}
