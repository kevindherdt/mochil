package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive

/** Failed/cancelled refreshes must not erase the currently displayed backlinks. */
internal suspend fun loadWikiBacklinks(
    load: suspend () -> List<WikiEntry>,
    onLoaded: (List<WikiEntry>) -> Unit,
) {
    val entries = try {
        load()
    } catch (cancelled: CancellationException) {
        throw cancelled
    } catch (_: Exception) {
        currentCoroutineContext().ensureActive()
        return
    }
    currentCoroutineContext().ensureActive()
    onLoaded(entries)
}
