package net.kdhmedia.mochil.playback

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.launch

/** One sender plus bounded latest-per-item work; callers own any durable recovery. */
internal class PlaybackProgressDeliveryQueue(
    scope: CoroutineScope,
    private val capacity: Int = 256,
    private val onFailure: (Exception) -> Unit = {},
) {
    init { require(capacity > 0) }
    private val lock = Any()
    private val pending = LinkedHashMap<String, suspend () -> Unit>()
    private val wake = Channel<Unit>(Channel.CONFLATED)
    private val worker = scope.launch(Dispatchers.IO) {
        try {
            for (signal in wake) {
                while (true) {
                    currentCoroutineContext().ensureActive()
                    val work = synchronized(lock) {
                        pending.entries.firstOrNull()?.let { entry -> pending.remove(entry.key) }
                    } ?: break
                    try { work() }
                    catch (cancelled: CancellationException) { throw cancelled }
                    catch (failure: Exception) { onFailure(failure) }
                }
            }
        } finally {
            synchronized(lock) { pending.clear(); wake.close() }
        }
    }

    fun submit(key: String, mayQueue: () -> Boolean = { true }, work: suspend () -> Unit): Boolean = synchronized(lock) {
        // Check while holding the queue lock: late producers must not overwrite newer work.
        if (!worker.isActive || !mayQueue()) return false
        pending[key] = work
        while (pending.size > capacity) pending.remove(pending.keys.first())
        wake.trySend(Unit).isSuccess
    }
}
