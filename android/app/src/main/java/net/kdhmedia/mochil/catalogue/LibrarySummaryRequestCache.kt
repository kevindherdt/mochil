package net.kdhmedia.mochil.catalogue

import java.util.concurrent.CompletableFuture
import java.util.concurrent.ExecutionException
import net.kdhmedia.mochil.auth.DeviceCredential

/** Coalesces short-lived duplicate library-summary reads without retaining device tokens. */
internal class LibrarySummaryRequestCache(
    private val nowMillis: () -> Long = System::currentTimeMillis,
    private val ttlMillis: Long = 5_000,
) {
    private data class Key(val serverUrl: String, val deviceId: String)
    private data class Cached(val values: List<LibrarySummary>, val expiresAt: Long)
    private val lock = Any()
    private val cached = mutableMapOf<Key, Cached>()
    private val inFlight = mutableMapOf<Key, CompletableFuture<List<LibrarySummary>>>()

    fun load(credential: DeviceCredential, force: Boolean = false, request: () -> List<LibrarySummary>): List<LibrarySummary> {
        val key = Key(credential.serverUrl, credential.deviceId)
        var leader = false
        val future = synchronized(lock) {
            if (!force) cached[key]?.takeIf { it.expiresAt > nowMillis() }?.let { return it.values }
            inFlight[key] ?: CompletableFuture<List<LibrarySummary>>().also {
                inFlight[key] = it
                leader = true
            }
        }
        if (leader) {
            try {
                val values = request()
                synchronized(lock) {
                    cached[key] = Cached(values, nowMillis() + ttlMillis)
                    inFlight.remove(key)
                }
                future.complete(values)
            } catch (failure: Throwable) {
                synchronized(lock) { inFlight.remove(key) }
                future.completeExceptionally(failure)
            }
        }
        return try {
            future.get()
        } catch (failure: ExecutionException) {
            throw failure.cause ?: failure
        }
    }
}
