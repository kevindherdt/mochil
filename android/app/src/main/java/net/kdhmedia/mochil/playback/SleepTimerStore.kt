package net.kdhmedia.mochil.playback

import android.content.Context

class SleepTimerStore(context: Context) {
    private val preferences = context.getSharedPreferences("playback_sleep_timer", Context.MODE_PRIVATE)

    fun set(minutes: Int, now: Long = System.currentTimeMillis()) {
        require(minutes in ALLOWED_MINUTES)
        preferences.edit().putLong(EXPIRES_AT, now + minutes * 60_000L).apply()
    }

    fun clear() { preferences.edit().remove(EXPIRES_AT).apply() }

    fun remainingMillis(now: Long = System.currentTimeMillis()): Long =
        (preferences.getLong(EXPIRES_AT, 0L) - now).coerceAtLeast(0L)

    fun consumeExpired(now: Long = System.currentTimeMillis()): Boolean {
        val expiresAt = preferences.getLong(EXPIRES_AT, 0L)
        if (expiresAt == 0L || expiresAt > now) return false
        clear()
        return true
    }

    companion object {
        val ALLOWED_MINUTES = setOf(15, 30, 45, 60, 90)
        private const val EXPIRES_AT = "expires_at"
    }
}

internal val spokenPlaybackSpeeds = listOf(1f, 1.25f, 1.5f, 1.75f, 2f)

/** Pause also clears play intent while buffering or temporarily audio-focus suppressed. */
internal fun pauseForExpiredSleepTimer(consumeExpired: () -> Boolean, pause: () -> Unit) {
    if (consumeExpired()) pause()
}

internal fun nextSpokenPlaybackSpeed(current: Float): Float {
    val index = spokenPlaybackSpeeds.indexOfFirst { kotlin.math.abs(it - current) < 0.01f }
    return spokenPlaybackSpeeds[(index + 1).mod(spokenPlaybackSpeeds.size)]
}
