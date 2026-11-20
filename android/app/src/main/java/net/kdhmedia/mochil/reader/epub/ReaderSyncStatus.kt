package net.kdhmedia.mochil.reader.epub

internal enum class ReaderSyncPhase { Synced, Syncing, DeviceOnly }

internal data class ReaderSyncStatus(
    val phase: ReaderSyncPhase = ReaderSyncPhase.DeviceOnly,
    val lastSynchronizedAtMillis: Long = 0,
)

internal fun readerSyncLabel(status: ReaderSyncStatus, formatTime: (Long) -> String): String = when (status.phase) {
    ReaderSyncPhase.Synced -> status.lastSynchronizedAtMillis.takeIf { it > 0 }
        ?.let { "Synced · ${formatTime(it)}" }
        ?: "Synced"
    ReaderSyncPhase.Syncing -> "Syncing…"
    ReaderSyncPhase.DeviceOnly -> status.lastSynchronizedAtMillis.takeIf { it > 0 }
        ?.let { "Saved on device · Last sync ${formatTime(it)}" }
        ?: "Saved on device"
}
