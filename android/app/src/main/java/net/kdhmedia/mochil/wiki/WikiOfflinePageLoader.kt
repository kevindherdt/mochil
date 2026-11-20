package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.withContext

/** Keep only the cacheable prefix plus one network batch, not the entire vault. */
internal suspend fun loadBoundedWikiPages(
    snapshot: WikiOfflineSnapshot,
    targets: List<WikiEntry>,
    batchSize: Int,
    loadBatch: suspend (List<WikiEntry>) -> List<WikiOfflinePage>,
    progress: (Int, Int) -> Unit,
): List<WikiOfflinePage> {
    require(batchSize in 1..4)
    currentCoroutineContext().ensureActive()
    val retained = mutableListOf<WikiOfflinePage>()
    progress(0, targets.size)
    val budget = withContext(Dispatchers.Default) { WikiOfflinePageBudget(snapshot) }
    for (batch in targets.take(MAX_OFFLINE_PAGES).chunked(batchSize)) {
        currentCoroutineContext().ensureActive()
        val loaded = loadBatch(batch)
        val accepted = withContext(Dispatchers.Default) {
            val context = currentCoroutineContext()
            loaded.takeWhile { context.ensureActive(); budget.accept(it) }
                .map { it.copy(page = it.page.copy(writable = false)) }
        }
        retained.addAll(accepted)
        progress(retained.size, targets.size)
        if (accepted.size < loaded.size || retained.size >= MAX_OFFLINE_PAGES) break
    }
    return retained.toList()
}
