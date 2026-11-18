package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive

internal class WikiMutationRefreshException(action: String, cause: Exception) : Exception(
    "$action, but the updated vault could not be loaded. Refresh the vault; do not repeat the completed operation.", cause,
)

/** Call only after the server has acknowledged the mutation. Never repeat the write. */
internal suspend fun <T> afterWikiMutation(action: String, refresh: suspend () -> T): T {
    currentCoroutineContext().ensureActive()
    return try {
        refresh().also { currentCoroutineContext().ensureActive() }
    } catch (cancelled: CancellationException) {
        throw cancelled
    } catch (failure: Exception) {
        currentCoroutineContext().ensureActive()
        throw WikiMutationRefreshException(action, failure)
    }
}
