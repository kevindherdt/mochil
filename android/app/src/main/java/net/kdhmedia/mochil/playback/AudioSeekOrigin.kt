package net.kdhmedia.mochil.playback

/** Prevents a server-derived resume seek from masquerading as viewer intent. */
internal class AudioSeekOrigin {
    private val consumed = object : LinkedHashMap<String, Unit>(16, .75f, true) {
        override fun removeEldestEntry(eldest: MutableMap.MutableEntry<String, Unit>?): Boolean = size > MAX_TOKENS
    }

    @Synchronized
    fun isExplicit(expected: Long, token: String, positionMs: Long): Boolean {
        if (expected <= 0L || token.isBlank() || consumed.containsKey(token)) return true
        consumed[token] = Unit
        return kotlin.math.abs(expected - positionMs.coerceAtLeast(0L)) > POSITION_TOLERANCE_MS
    }

    private companion object {
        const val POSITION_TOLERANCE_MS = 1_000L
        const val MAX_TOKENS = 64
    }
}
