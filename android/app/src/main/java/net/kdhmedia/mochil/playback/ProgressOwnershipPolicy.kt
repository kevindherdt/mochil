package net.kdhmedia.mochil.playback

/**
 * Shared revision-conflict policy for passive media progress writers.
 * A stale player may advance an item, but it must never rewind or unfinish the
 * newer state produced by another device. Equivalent clock samples are already
 * synchronized and must not create a fresh revision/device attribution.
 */
internal fun shouldUploadPassiveProgress(
    remotePosition: Double,
    remoteFinished: Boolean,
    localPosition: Double,
    duration: Double,
    localFinished: Boolean = false,
): Boolean {
    if (!remotePosition.isFinite() || !localPosition.isFinite() || !duration.isFinite() || duration <= 0.0) return false
    if (remoteFinished) return false
    val boundedLocal = localPosition.coerceIn(0.0, duration)
    if (localFinished) return boundedLocal + POSITION_EQUIVALENCE_SECONDS >= remotePosition
    return boundedLocal > remotePosition + POSITION_EQUIVALENCE_SECONDS
}

/** Periodic writers exist only while media is actively advancing. */
internal fun shouldPersistPeriodicProgress(isPlaying: Boolean, hasCurrentItem: Boolean): Boolean =
    isPlaying && hasCurrentItem

/** Mirrors the server's canonical 98% audio-completion boundary. */
internal fun audioProgressFinished(position: Double, duration: Double): Boolean =
    position.isFinite() && duration.isFinite() && duration > 0.0 &&
        position.coerceIn(0.0, duration) / duration >= AUDIO_FINISHED_FRACTION

private const val POSITION_EQUIVALENCE_SECONDS = 0.25
private const val AUDIO_FINISHED_FRACTION = 0.98
