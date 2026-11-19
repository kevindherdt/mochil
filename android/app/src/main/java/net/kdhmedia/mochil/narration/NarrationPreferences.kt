package net.kdhmedia.mochil.narration

import android.content.Context

/** The voice and speed the reader last used, kept on this device. */
class NarrationPreferences(context: Context) {
    private val preferences = context.getSharedPreferences("narration", Context.MODE_PRIVATE)

    fun voice(): String = preferences.getString("voice", "").orEmpty()
    fun saveVoice(id: String) = preferences.edit().putString("voice", id).apply()

    fun speed(): Float = preferences.getFloat("speed", 1f).takeIf { it in narrationSpeeds } ?: 1f
    fun saveSpeed(speed: Float) = preferences.edit().putFloat("speed", speed).apply()

    /** Speed for a book's own recording, which can go faster than the server's voice. */
    fun recordingSpeed(): Float = preferences.getFloat("recording_speed", 1f).takeIf { it in recordingSpeeds } ?: 1f
    fun saveRecordingSpeed(speed: Float) = preferences.edit().putFloat("recording_speed", speed).apply()

    /** The book being read aloud ("library/book"), so a reader that opens it later picks the reading up; empty when none. */
    fun activeBook(): String = preferences.getString("active_book", "").orEmpty()
    fun saveActiveBook(key: String) = preferences.edit().putString("active_book", key).apply()
}
