package net.kdhmedia.mochil.video

import android.os.SystemClock
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.playback.PlaybackProgressDeliveryQueue

/** Owns submission ordering while borrowing the player's existing lifecycle scope. */
internal class VideoProgressReporter(
    private val scope: CoroutineScope,
    private val now: () -> Long = SystemClock::elapsedRealtime,
    private val epochNow: () -> Long = System::currentTimeMillis,
    private val journal: VideoProgressJournal? = null,
    private val onPersistenceFailure: () -> Unit = {},
) {
    private val client = VideoProgressClient()
    private val submissions = VideoProgressSubmissions()
    private val delivery = PlaybackProgressDeliveryQueue(scope)

    fun submit(credential: DeviceCredential, entry: VideoPlaybackEntry,
               position: Long, duration: Long, finished: Boolean, explicitSeek: Boolean) {
        if (duration <= 0) return
        if (untouchedFinishedReplay(entry, position.toDouble(), finished, explicitSeek)) return
        val epoch = epochNow()
        val monotonic = now()
        val checkpoint = PendingVideoProgress(
            videoProgressOwner(credential), entry.libraryId, entry.itemId,
            position.coerceIn(0, duration) / 1_000.0, duration / 1_000.0,
            entry.progressRevision.coerceAtLeast(0), finished, if (explicitSeek) epoch else 0, entry.revision, entry.finished,
        )
        val committed = journal?.put(checkpoint)
        if (journal != null && committed == null) onPersistenceFailure()
        val pendingSeek = committed?.explicitSeekForReplay(epoch) == true
        val ticket = submissions.submit(
            videoProgressCacheKey(credential, entry.libraryId, entry.itemId),
            explicitSeek || pendingSeek, monotonic,
            seekAt = if (!explicitSeek && pendingSeek) monotonic - (epoch - committed.explicitSeekAtEpochMs) else monotonic,
            committed = committed,
        )
        enqueue(credential, entry, checkpoint.positionSeconds, checkpoint.durationSeconds, finished, ticket)
    }

    fun restore(credential: DeviceCredential, queue: VideoPlaybackQueue): VideoPlaybackQueue =
        runCatching { restorePendingVideoPosition(queue, journal?.entries().orEmpty(), videoProgressOwner(credential), epochNow()) }
            .getOrElse { onPersistenceFailure(); queue }

    fun replay(credential: DeviceCredential) = scope.launch(Dispatchers.IO) {
        val owner = videoProgressOwner(credential)
        val pending = runCatching { journal?.entries().orEmpty() }.getOrElse {
            onPersistenceFailure()
            return@launch
        }
        for (value in pending) {
            currentCoroutineContext().ensureActive()
            if (value.owner != owner) continue
            val epoch = epochNow()
            val monotonic = now()
            val seek = value.explicitSeekForReplay(epoch)
            val ticket = submissions.replay(
                videoProgressCacheKey(credential, value.libraryId, value.itemId), seek, monotonic,
                if (seek) monotonic - (epoch - value.explicitSeekAtEpochMs) else monotonic, value,
            ) ?: continue
            // The progress client needs identity/revision only, never a replay stream URL.
            val entry = VideoPlaybackEntry(value.libraryId, value.itemId, "", "", "", 0,
                value.durationSeconds, "", revision = value.mediaRevision, progressRevision = value.rebasedFromRevision ?: value.revision, finished = value.replayFinished)
            enqueue(credential, entry, value.positionSeconds, value.durationSeconds, value.finished, ticket)
        }
    }

    private fun enqueue(credential: DeviceCredential, entry: VideoPlaybackEntry, position: Double,
                        duration: Double, finished: Boolean, ticket: VideoProgressSubmissions.Ticket) {
        delivery.submit(ticket.key, mayQueue = { submissions.isCurrent(ticket) }) {
            var sentCheckpoint = ticket.committed
            var rebasePersisted = true
            if (client.save(credential, entry, position, duration, finished, ticket.seekActive(now()), initialRevision = ticket.committed?.revision ?: entry.progressRevision, onSaved = { sentRevision, revision ->
                    ticket.committed?.let {
                        val sent = if (it.revision == sentRevision) it else
                            it.copy(revision = sentRevision, rebasedFromRevision = it.rebasedFromRevision ?: it.revision)
                        sentCheckpoint = sent
                        if (journal?.rebaseNewer(sent, revision, it.revision) == false) {
                            rebasePersisted = false
                            onPersistenceFailure()
                        }
                    }
                }) { submissions.isCurrent(ticket) }) {
                submissions.acknowledge(ticket)
                if (rebasePersisted) sentCheckpoint?.let { if (journal?.removeIfCurrent(it) != true && it != ticket.committed) ticket.committed?.let { original -> journal?.removeIfCurrent(original) } }
            }
        }
    }
}
