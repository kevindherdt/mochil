package net.kdhmedia.mochil.playback

/** Return the exact committed record, including inherited unsent seek intent. */
internal fun commitPendingAudioProgress(
    current: List<PendingAudioProgress>,
    value: PendingAudioProgress,
    limit: Int,
    persist: (List<PendingAudioProgress>) -> Boolean,
): PendingAudioProgress? {
    val next = mergePendingAudioProgress(current.filter { it.owner == value.owner }, value, limit)
    val committed = next.lastOrNull() ?: return null
    return committed.takeIf { persist(next) }
}
