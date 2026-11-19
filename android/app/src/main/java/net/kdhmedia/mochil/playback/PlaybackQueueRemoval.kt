package net.kdhmedia.mochil.playback

/** Descending indices remain valid as earlier removals shrink the player queue. */
internal fun queuedCopyRemovalIndices(
    count: Int,
    currentIndex: Int,
    additions: Set<String>,
    keyAt: (Int) -> String,
): List<Int> = if (additions.isEmpty()) emptyList() else
    (count - 1 downTo 0).filter { it != currentIndex && keyAt(it) in additions }

/** Half-open ranges, highest first, matching Player.removeMediaItems semantics. */
internal data class QueueRemovalRange(val start: Int, val endExclusive: Int)

internal fun queuedCopyRemovalRanges(
    count: Int,
    currentIndex: Int,
    additions: Set<String>,
    keyAt: (Int) -> String,
): List<QueueRemovalRange> {
    val ranges = mutableListOf<QueueRemovalRange>()
    for (index in queuedCopyRemovalIndices(count, currentIndex, additions, keyAt)) {
        val previous = ranges.lastOrNull()
        if (previous != null && index + 1 == previous.start) {
            ranges[ranges.lastIndex] = previous.copy(start = index)
        } else {
            ranges += QueueRemovalRange(index, index + 1)
        }
    }
    return ranges
}
