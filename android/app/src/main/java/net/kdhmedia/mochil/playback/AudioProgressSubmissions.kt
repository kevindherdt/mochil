package net.kdhmedia.mochil.playback

/** Small, non-network lock: newer local intent supersedes queued sends for the same item. */
internal class AudioProgressSubmissions(private val capacity: Int = 256) {
    init { require(capacity > 0) }
    internal class Ticket internal constructor(internal val key: String, val value: PendingAudioProgress, val committed: PendingAudioProgress?) {
        internal var acknowledged = false
    }
    private val latest = LinkedHashMap<String, Ticket>()

    @Synchronized fun submit(value: PendingAudioProgress, replay: Boolean = false, committed: PendingAudioProgress? = null): Ticket? {
        val key = "${value.owner}\u0000${value.libraryId}\u0000${value.itemId}"
        if (replay && latest.containsKey(key)) return null
        val pending = latest[key]?.takeUnless { it.acknowledged }?.value
        val merged = mergePendingAudioProgress(listOfNotNull(pending), value, 1).single()
        val ticket = Ticket(key, merged, committed ?: latest[key]?.committed)
        latest.remove(key)
        latest[key] = ticket
        while (latest.size > capacity) latest.remove(latest.keys.first())
        return ticket
    }

    @Synchronized fun isCurrent(ticket: Ticket): Boolean = latest[ticket.key] === ticket

    @Synchronized fun acknowledge(ticket: Ticket) {
        if (latest[ticket.key] === ticket) ticket.acknowledged = true
    }
}
