package net.kdhmedia.mochil.wiki

/** Session-local acknowledgement barrier. An older in-flight read cannot clear it. */
internal class WikiSaveRecovery {
    internal class Ticket(val failure: WikiSavedButReloadFailedException)
    private val pending = mutableMapOf<List<String>, Ticket>()

    @Synchronized fun ticket(key: List<String>): Ticket? = pending[key]

    @Synchronized fun requireFresh(key: List<String>) {
        pending[key]?.let { throw it.failure }
    }

    @Synchronized fun record(key: List<String>, failure: WikiSavedButReloadFailedException) {
        pending[key] = Ticket(failure)
    }

    @Synchronized fun refreshed(key: List<String>, ticket: Ticket?) {
        if (ticket != null && pending[key] === ticket) pending.remove(key)
    }
}
