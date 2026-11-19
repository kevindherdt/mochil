package net.kdhmedia.mochil.video

/** Bounded local intent ordering; network calls never hold this lock. */
internal class VideoProgressSubmissions(private val capacity: Int = 256) {
    init { require(capacity > 0) }
    internal class Ticket(val key: String, val explicitSeek: Boolean, val seekAt: Long, val committed: PendingVideoProgress?) {
        var acknowledged = false
        fun seekActive(now: Long) = explicitSeek && now >= seekAt && now - seekAt <= 600_000
    }
    private val latest = LinkedHashMap<String, Ticket>()

    @Synchronized fun submit(key: String, explicitSeek: Boolean, now: Long, seekAt: Long = now, committed: PendingVideoProgress? = null): Ticket {
        val previous = latest[key]?.takeUnless { it.acknowledged }
        val retainedSeek = previous?.explicitSeek == true && now >= previous.seekAt && now - previous.seekAt <= 600_000
        val ticket = Ticket(key, explicitSeek || retainedSeek, if (explicitSeek) seekAt else previous?.seekAt ?: now, committed ?: previous?.committed)
        latest.remove(key)
        latest[key] = ticket
        while (latest.size > capacity) latest.remove(latest.keys.first())
        return ticket
    }

    @Synchronized fun isCurrent(ticket: Ticket) = latest[ticket.key] === ticket

    @Synchronized fun replay(key: String, explicitSeek: Boolean, now: Long, seekAt: Long, committed: PendingVideoProgress): Ticket? {
        if (latest.containsKey(key)) return null
        return submit(key, explicitSeek, now, seekAt, committed)
    }

    @Synchronized fun acknowledge(ticket: Ticket) {
        if (latest[ticket.key] === ticket) ticket.acknowledged = true
    }
}
