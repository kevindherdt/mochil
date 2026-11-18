package net.kdhmedia.mochil.video

/** Recover only the selected item; never replace media URLs or another queue item's state. */
internal fun restorePendingVideoPosition(
    queue: VideoPlaybackQueue,
    pending: List<PendingVideoProgress>,
    owner: String,
    nowEpochMs: Long,
): VideoPlaybackQueue {
    val entry = queue.entries.getOrNull(queue.currentIndex) ?: return queue
    val value = pending.lastOrNull {
        it.valid() && it.owner == owner && it.libraryId == entry.libraryId && it.itemId == entry.itemId &&
            it.mediaRevision == entry.revision
    } ?: return queue
    // A completed checkpoint is synchronization work, not an instruction to reopen at EOF.
    if (value.finished) return queue
    val explicit = value.explicitSeekForReplay(nowEpochMs)
    val sameReplay = entry.finished && value.replayFinished && value.revision == entry.progressRevision
    if (!explicit && !sameReplay && (entry.finished || value.positionSeconds * 1000 < entry.positionMs)) return queue
    val duration = entry.durationSeconds.takeIf { it.isFinite() && it > 0 } ?: value.durationSeconds
    val position = (value.positionSeconds.coerceAtMost(duration) * 1000).toLong().coerceAtLeast(0)
    // The entry remains the remote baseline used by the progress client. The
    // checkpoint controls playback only until it is actually acknowledged.
    return queue.copy(startOverrideMs = position)
}
