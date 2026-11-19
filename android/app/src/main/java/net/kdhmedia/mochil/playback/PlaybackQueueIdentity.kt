package net.kdhmedia.mochil.playback

/** Resolve a menu's captured identity against the current player, never its old row. */
internal fun playbackQueueIndex(count: Int, identity: String, keyAt: (Int) -> String): Int =
    (0 until count).firstOrNull { keyAt(it) == identity } ?: -1
