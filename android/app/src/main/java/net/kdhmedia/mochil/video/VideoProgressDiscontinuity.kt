package net.kdhmedia.mochil.video

import androidx.media3.common.C
import androidx.media3.common.Player

internal data class VideoProgressCheckpoint(
    val entry: VideoPlaybackEntry,
    val positionMs: Long,
    val durationMs: Long,
    val finished: Boolean,
    val explicitSeek: Boolean,
)

internal fun videoProgressCheckpoint(
    oldPosition: Player.PositionInfo,
    newPosition: Player.PositionInfo,
    reason: Int,
    entries: List<VideoPlaybackEntry>,
    playerDurationMs: Long,
    seekOrigin: VideoSeekOrigin,
): VideoProgressCheckpoint? {
    if (oldPosition.mediaItemIndex != newPosition.mediaItemIndex) {
        val previous = entries.getOrNull(oldPosition.mediaItemIndex) ?: return null
        val duration = previous.durationSeconds.takeIf { it > 0 }?.times(1_000)?.toLong()
            ?: oldPosition.positionMs
        return VideoProgressCheckpoint(
            previous,
            oldPosition.positionMs,
            duration,
            reason == Player.DISCONTINUITY_REASON_AUTO_TRANSITION,
            explicitSeek = false,
        )
    }
    if (reason != Player.DISCONTINUITY_REASON_SEEK) return null
    val current = entries.getOrNull(newPosition.mediaItemIndex) ?: return null
    val duration = playerDurationMs.takeIf { it != C.TIME_UNSET && it > 0 }
        ?: current.durationSeconds.takeIf { it > 0 }?.times(1_000)?.toLong()
        ?: return null
    return VideoProgressCheckpoint(
        current,
        newPosition.positionMs,
        duration,
        finished = false,
        explicitSeek = seekOrigin.isExplicit(newPosition.mediaItemIndex, newPosition.positionMs),
    )
}
