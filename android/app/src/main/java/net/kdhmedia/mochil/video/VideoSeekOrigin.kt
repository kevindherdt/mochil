package net.kdhmedia.mochil.video

/**
 * Distinguishes recovery seeks initiated by the player from deliberate viewer seeks.
 *
 * Media3 reports both through DISCONTINUITY_REASON_SEEK. Only deliberate viewer
 * seeks may override newer progress saved by another device.
 */
internal class VideoSeekOrigin {
    private var pending: Target? = null

    fun expectAutomatic(mediaItemIndex: Int, positionMs: Long) {
        pending = Target(mediaItemIndex, positionMs.coerceAtLeast(0L))
    }

    fun isExplicit(mediaItemIndex: Int, positionMs: Long): Boolean {
        val expected = pending ?: return true
        pending = null
        return expected.mediaItemIndex != mediaItemIndex ||
            kotlin.math.abs(expected.positionMs - positionMs.coerceAtLeast(0L)) > POSITION_TOLERANCE_MS
    }

    fun clear() {
        pending = null
    }

    private data class Target(val mediaItemIndex: Int, val positionMs: Long)

    private companion object {
        const val POSITION_TOLERANCE_MS = 1_000L
    }
}
