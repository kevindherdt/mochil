package net.kdhmedia.mochil.wiki

import java.util.concurrent.atomic.AtomicReference

/** Intent order is recorded before dispatch; storage completion order cannot override it. */
internal class WikiDraftWriteGate {
    private data class Intent(val ticket: Any = Any(), val readable: Boolean = true)
    private val latest = AtomicReference(Intent())
    private val storageLock = Any()

    fun begin(): Any = latest.updateAndGet { Intent(readable = it.readable) }.ticket

    fun write(ticket: Any, operation: () -> Unit) = synchronized(storageLock) {
        if (latest.get().ticket === ticket) {
            operation()
            latest.updateAndGet { if (it.ticket === ticket) it.copy(readable = true) else it }
        }
    }

    fun clear(operation: () -> Unit) {
        clearLater({ it() }, operation)
    }

    fun clearLater(dispatch: (() -> Unit) -> Unit, operation: () -> Unit) {
        val intent = Intent(readable = false)
        latest.set(intent)
        dispatch {
            synchronized(storageLock) {
                if (latest.get().ticket === intent.ticket) operation()
            }
        }
    }

    fun <T> readVisible(operation: () -> T): T? = synchronized(storageLock) {
        val started = latest.get()
        if (!started.readable) return@synchronized null
        val result = operation()
        // Clear/new-edit intent does not acquire storageLock, so it may arrive
        // while decryption is running. Never publish that superseded recovery.
        val finished = latest.get()
        if (finished.readable && finished.ticket === started.ticket) result else null
    }

    fun <T> read(operation: () -> T): T = synchronized(storageLock) { operation() }
}
