package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive

/** Missing optional artwork is recoverable; cancellation is not a missing image. */
internal suspend fun loadOptionalWikiAsset(load: () -> ByteArray): ByteArray? {
    currentCoroutineContext().ensureActive()
    val bytes = try { load() }
    catch (cancelled: CancellationException) { throw cancelled }
    catch (_: Exception) { null }
    currentCoroutineContext().ensureActive()
    return bytes
}

/** Keeps only a bounded batch beyond retained bytes, and stops once full. */
internal suspend fun loadBoundedWikiAssets(
    targets: List<WikiEntry>,
    batchSize: Int,
    loadBatch: suspend (List<WikiEntry>) -> List<Pair<WikiEntry, ByteArray?>>,
): Map<WikiEntry, ByteArray> {
    require(batchSize in 1..4)
    val retained = linkedMapOf<WikiEntry, ByteArray>()
    var total = 0
    for (batch in targets.distinct().take(MAX_OFFLINE_ASSETS).chunked(batchSize)) {
        currentCoroutineContext().ensureActive()
        if (total >= MAX_OFFLINE_ASSET_TOTAL_BYTES) break
        val loaded = loadBatch(batch)
        currentCoroutineContext().ensureActive()
        for ((entry, bytes) in loaded) {
            if (entry !in batch || entry in retained || bytes == null || bytes.isEmpty() || bytes.size > MAX_OFFLINE_ASSET_BYTES) continue
            if (total + bytes.size > MAX_OFFLINE_ASSET_TOTAL_BYTES) continue
            retained[entry] = bytes
            total += bytes.size
        }
    }
    return retained
}
