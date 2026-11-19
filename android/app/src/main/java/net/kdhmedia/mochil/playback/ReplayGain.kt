package net.kdhmedia.mochil.playback

import android.content.Context
import androidx.media3.common.C
import androidx.media3.common.audio.AudioProcessor
import androidx.media3.common.audio.BaseAudioProcessor
import androidx.media3.common.util.UnstableApi
import java.nio.ByteBuffer
import kotlin.math.min
import kotlin.math.pow

enum class ReplayGainMode { Off, Track, Album }

class ReplayGainPreferences(context: Context) {
    private val preferences = context.getSharedPreferences(NAME, Context.MODE_PRIVATE)
    fun mode(): ReplayGainMode = runCatching { ReplayGainMode.valueOf(preferences.getString(MODE, ReplayGainMode.Off.name).orEmpty()) }.getOrDefault(ReplayGainMode.Off)
    fun setMode(mode: ReplayGainMode) = preferences.edit().putString(MODE, mode.name).commit()
    fun sharedPreferences() = preferences

    companion object { const val NAME = "playback_preferences"; const val MODE = "replay_gain_mode" }
}

data class ReplayGainMetadata(
    val trackGainDb: Double? = null,
    val albumGainDb: Double? = null,
    val trackPeak: Double? = null,
    val albumPeak: Double? = null,
)

internal object ReplayGainCalculator {
    fun multiplier(mode: ReplayGainMode, metadata: ReplayGainMetadata): Float {
        if (mode == ReplayGainMode.Off) return 1f
        val gain = when (mode) {
            ReplayGainMode.Track -> metadata.trackGainDb ?: metadata.albumGainDb
            ReplayGainMode.Album -> metadata.albumGainDb ?: metadata.trackGainDb
            ReplayGainMode.Off -> null
        } ?: return 1f
        val peak = when (mode) {
            ReplayGainMode.Track -> metadata.trackPeak ?: metadata.albumPeak
            ReplayGainMode.Album -> metadata.albumPeak ?: metadata.trackPeak
            ReplayGainMode.Off -> null
        }
        val requested = 10.0.pow(gain.coerceIn(-60.0, 30.0) / 20.0)
        val protected = if (peak != null && peak.isFinite() && peak > 0) min(requested, 1.0 / peak) else requested
        return protected.coerceIn(0.001, 32.0).toFloat()
    }
}

@UnstableApi
class ReplayGainAudioProcessor : BaseAudioProcessor() {
    @Volatile private var multiplier = 1f

    fun setMultiplier(value: Float) { multiplier = value.coerceIn(0.001f, 32f) }

    override fun onConfigure(inputAudioFormat: AudioProcessor.AudioFormat): AudioProcessor.AudioFormat {
        if (inputAudioFormat.encoding != C.ENCODING_PCM_16BIT && inputAudioFormat.encoding != C.ENCODING_PCM_FLOAT) {
            throw AudioProcessor.UnhandledAudioFormatException(inputAudioFormat)
        }
        return inputAudioFormat
    }

    override fun queueInput(inputBuffer: ByteBuffer) {
        val output = replaceOutputBuffer(inputBuffer.remaining())
        output.order(inputBuffer.order())
        if (inputAudioFormat.encoding == C.ENCODING_PCM_FLOAT) {
            while (inputBuffer.remaining() >= Float.SIZE_BYTES) {
                output.putFloat((inputBuffer.float * multiplier).coerceIn(-1f, 1f))
            }
        } else {
            while (inputBuffer.remaining() >= Short.SIZE_BYTES) {
                val sample = (inputBuffer.short.toInt() * multiplier).toInt().coerceIn(Short.MIN_VALUE.toInt(), Short.MAX_VALUE.toInt())
                output.putShort(sample.toShort())
            }
        }
        inputBuffer.position(inputBuffer.limit())
        output.flip()
    }
}
