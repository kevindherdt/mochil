package net.kdhmedia.mochil.playback

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive

/** Cancellation is not a failed send: stop replay and retain the unacknowledged outbox. */
internal suspend fun replayPendingAudioProgress(
    entries: List<PendingAudioProgress>,
    send: (PendingAudioProgress) -> Boolean,
    acknowledge: (PendingAudioProgress) -> Unit,
) {
    currentCoroutineContext().ensureActive()
    for (entry in entries) {
        currentCoroutineContext().ensureActive()
        val saved = try { send(entry) }
        catch (cancelled: CancellationException) { throw cancelled }
        catch (_: Exception) { false }
        currentCoroutineContext().ensureActive()
        if (saved) acknowledge(entry)
    }
}
