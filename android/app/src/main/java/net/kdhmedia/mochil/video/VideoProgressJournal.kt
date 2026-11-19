package net.kdhmedia.mochil.video

import android.content.Context
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import org.json.JSONObject

/** Synchronous commit is explicit: null/false never promises a durable checkpoint. */
internal class VideoProgressJournal(
    private val owner: String,
    private val load: () -> ByteArray?,
    private val persist: (ByteArray) -> Boolean,
) {
    fun entries(): List<PendingVideoProgress> = synchronized(journalLock) { read().pending }

    fun put(value: PendingVideoProgress): PendingVideoProgress? = synchronized(journalLock) {
        if (value.owner != owner || !value.valid()) return@synchronized null
        runCatching {
            val state = read()
            val ack = state.acknowledged.lastOrNull { sameLineage(it, value) && it.rebasedFromRevision == value.revision }
            val rebased = if (ack != null) value.copy(revision = ack.revision, rebasedFromRevision = ack.rebasedFromRevision) else value
            val merged = mergePendingVideoProgress(state.pending, rebased)
            if (write(state.copy(pending = merged))) merged.last() else null
        }.getOrNull()
    }

    // Only rebase a later checkpoint derived from the exact revision we saved.
    fun rebaseNewer(saved: PendingVideoProgress, revision: Long, checkpointRevision: Long = saved.revision): Boolean = synchronized(journalLock) {
        runCatching {
            if (saved.owner != owner || !saved.valid() || revision < saved.revision) return@synchronized false
            val state = read()
            val previous = state.acknowledged.lastOrNull { sameLineage(it, saved) }
            if (previous != null && saved.revision < previous.revision) return@synchronized true
            val later = state.pending.firstOrNull {
                it.owner == saved.owner && it.libraryId == saved.libraryId && it.itemId == saved.itemId &&
                    it.mediaRevision == saved.mediaRevision &&
                    (it.revision == saved.revision || it.revision == checkpointRevision) &&
                    it.replayFinished == saved.replayFinished &&
                    it.copy(revision = saved.revision, rebasedFromRevision = saved.rebasedFromRevision) != saved
            }
            val ack = saved.copy(revision = revision, rebasedFromRevision = saved.rebasedFromRevision ?: saved.revision)
            write(State(state.pending.map {
                if (it == later) it.copy(revision = revision, rebasedFromRevision = saved.rebasedFromRevision ?: saved.revision) else it
            }, (state.acknowledged.filterNot { sameLineage(it, saved) } + ack).takeLast(MAX_PENDING_VIDEO_PROGRESS)))
        }.getOrDefault(false)
    }

    fun removeIfCurrent(value: PendingVideoProgress): Boolean = synchronized(journalLock) {
        if (value.owner != owner) return@synchronized false
        runCatching {
            val state = read()
            val retained = state.pending.filterNot { it == value }
            if (retained.size == state.pending.size) false else write(state.copy(pending = retained))
        }.getOrDefault(false)
    }

    private data class State(val pending: List<PendingVideoProgress>, val acknowledged: List<PendingVideoProgress>)

    private fun read(): State {
        val bytes = load() ?: return State(emptyList(), emptyList())
        require(bytes.size <= MAX_VIDEO_PROGRESS_JOURNAL_BYTES)
        val text = bytes.decodeToString().trimStart()
        if (text.startsWith("[")) return State(decodePendingVideoProgressStrict(bytes).filter { it.owner == owner }, emptyList())
        val json = JSONObject(text)
        fun values(key: String) = decodePendingVideoProgressStrict(json.getJSONArray(key).toString().encodeToByteArray()).filter { it.owner == owner }
        return State(values("pending"), values("acknowledged"))
    }

    private fun write(state: State): Boolean {
        val json = JSONObject().put("pending", org.json.JSONArray(encodePendingVideoProgress(state.pending).decodeToString()))
            .put("acknowledged", org.json.JSONArray(encodePendingVideoProgress(state.acknowledged).decodeToString()))
            .toString().encodeToByteArray()
        require(json.size <= MAX_VIDEO_PROGRESS_JOURNAL_BYTES)
        return persist(json)
    }

    private fun sameLineage(a: PendingVideoProgress, b: PendingVideoProgress) =
        a.owner == b.owner && a.libraryId == b.libraryId && a.itemId == b.itemId &&
            a.mediaRevision == b.mediaRevision && a.replayFinished == b.replayFinished

    // Multiple player instances may briefly overlap during activity recreation.
    private companion object { val journalLock = Any() }
}

internal fun createVideoProgressJournal(context: Context, credential: DeviceCredential): VideoProgressJournal {
    val owner = videoProgressOwner(credential)
    val storage = EncryptedPayloadStore(
        context.applicationContext, "video_progress_outbox_$owner", "mochil_video_progress_outbox",
        maxPlaintextBytes = MAX_VIDEO_PROGRESS_JOURNAL_BYTES,
    )
    return VideoProgressJournal(owner, storage::loadStrict, storage::save)
}

internal fun videoProgressJournalOrNull(context: Context, credential: DeviceCredential): VideoProgressJournal? =
    runCatching { createVideoProgressJournal(context, credential) }.getOrElse {
        PerformanceDiagnostics.record("storage", "video_progress_unavailable")
        null
    }
