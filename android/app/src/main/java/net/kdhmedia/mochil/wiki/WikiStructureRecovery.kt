package net.kdhmedia.mochil.wiki

import kotlinx.coroutines.CancellationException

/** Expected tree evidence for an already acknowledged structural write. */
internal data class WikiStructureChange(
    val targetPath: String? = null,
    val folder: Boolean = false,
    val removedId: String? = null,
) {
    fun reflectedBy(entries: List<WikiEntry>): Boolean =
        (targetPath == null || entries.any { it.path == targetPath && it.isFolder == folder }) &&
            (removedId == null || entries.none { it.id == removedId })
}

internal class WikiStructureRecovery {
    internal class Ticket(val change: WikiStructureChange, val failure: Exception)
    private val pending = mutableMapOf<List<String>, Ticket>()

    @Synchronized fun requireFresh(key: List<String>) {
        pending[key]?.let { throw it.failure }
    }

    @Synchronized fun record(key: List<String>, change: WikiStructureChange, failure: Exception) {
        pending[key] = Ticket(change, failure)
    }

    @Synchronized fun tickets(owner: List<String>): Map<List<String>, Ticket> =
        pending.filterKeys { it.take(owner.size) == owner }

    @Synchronized fun refreshed(tickets: Map<List<String>, Ticket>, entries: List<WikiEntry>) {
        tickets.forEach { (key, ticket) ->
            if (pending[key] === ticket && ticket.change.reflectedBy(entries)) pending.remove(key)
        }
    }

    suspend fun <T> mutate(
        key: List<String>, change: WikiStructureChange, action: String,
        write: () -> Unit, refresh: suspend () -> T,
    ): T {
        requireFresh(key)
        try {
            write()
        } catch (failure: WikiException) {
            if (!failure.mutationCommitted) throw failure
            throw WikiMutationRefreshException(action, failure).also { record(key, change, it) }
        }
        return try {
            afterWikiMutation(action, refresh)
        } catch (cancelled: CancellationException) {
            // Cancellation stops further IO, not the already completed filesystem write.
            record(key, change, WikiMutationRefreshException(action, cancelled))
            throw cancelled
        } catch (failure: WikiMutationRefreshException) {
            record(key, change, failure)
            throw failure
        }
    }
}
