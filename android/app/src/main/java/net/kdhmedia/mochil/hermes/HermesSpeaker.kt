package net.kdhmedia.mochil.hermes

import android.media.AudioAttributes
import android.media.AudioFormat
import android.media.AudioTrack
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential

/**
 * Speaks her replies. Text arrives while the reply streams; it is cut into sentences, each sentence is fetched from the
 * server as streamed audio the moment it is known (so the next one is ready while the current one plays), and the audio
 * is written to one [AudioTrack] in order. Stopping cuts the sound at once and drops what was not yet played.
 */
class HermesSpeaker(
    private val client: HermesClient,
    private val credential: () -> DeviceCredential?,
    private val voice: () -> String?,
    private val onChange: (speaking: Boolean) -> Unit,
    private val onError: (Throwable) -> Unit = {},
    private val prebufferSeconds: Double = 0.25,
) {
    private class Sentence(val text: String) {
        val rate = CompletableDeferred<Int>()
        val pieces = Channel<ByteArray>(Channel.UNLIMITED)
    }

    private class Reply(val scope: CoroutineScope, val sentences: Channel<Sentence>, var buffer: String = "")

    @Volatile private var reply: Reply? = null
    @Volatile private var track: AudioTrack? = null
    @Volatile private var active = false

    val speaking: Boolean get() = active

    // Loudness of the audio written to the track, one value per 20 ms, so the avatar's mouth can follow what is
    // playing right now (the track's play position) rather than what has merely arrived.
    private val loudness = ArrayList<Float>()
    private var blockFrames = 480
    private var blockSum = 0.0
    private var blockCount = 0

    @Synchronized private fun measure(bytes: ByteArray) {
        var i = 0
        while (i + 1 < bytes.size) {
            val sample = ((bytes[i + 1].toInt() shl 8) or (bytes[i].toInt() and 0xff)).toShort() / 32768.0
            blockSum += sample * sample
            blockCount += 1
            if (blockCount >= blockFrames) { loudness += Math.sqrt(blockSum / blockCount).toFloat(); blockSum = 0.0; blockCount = 0 }
            i += 2
        }
    }

    @Synchronized private fun resetLoudness(rate: Int) { loudness.clear(); blockSum = 0.0; blockCount = 0; blockFrames = maxOf(1, rate / 50) }

    /** How loud the sound playing right now is, 0 to 1; 0 when nothing plays. */
    @Synchronized fun level(): Float {
        val head = runCatching { track?.playbackHeadPosition }.getOrNull() ?: return 0f
        return loudness.getOrNull(head / blockFrames) ?: 0f
    }

    private fun setActive(value: Boolean) { if (active != value) { active = value; onChange(value) } }

    /** Starts a new reply, silencing any earlier one. Feed it with [push] and finish with [end]. */
    fun begin() {
        stop()
        val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
        val current = Reply(scope, Channel(Channel.UNLIMITED))
        reply = current
        setActive(true)
        scope.launch { play(current) }
    }

    fun push(text: String) {
        val current = reply ?: return
        current.buffer += text
        val taken = takeSentences(current.buffer)
        current.buffer = taken.rest
        taken.sentences.forEach { enqueue(current, it) }
    }

    fun end() {
        val current = reply ?: return
        takeSentences(current.buffer, final = true).sentences.forEach { enqueue(current, it) }
        current.buffer = ""
        current.sentences.close()
    }

    /** Reads a finished message from the start. */
    fun speakText(text: String) { begin(); push(text); end() }

    fun stop() {
        val old = reply ?: return
        reply = null
        old.scope.cancel()
        old.sentences.close()
        release()
        resetLoudness(24000)
        setActive(false)
    }

    private fun enqueue(current: Reply, text: String) {
        val sentence = Sentence(text)
        current.sentences.trySend(sentence)
        current.scope.launch {
            val credential = credential()
            try {
                if (credential == null) throw HermesException("This device is not signed in")
                client.speak(credential, text, voice(), { sentence.rate.complete(it) }, { sentence.pieces.trySend(it) })
            } catch (cancelled: kotlinx.coroutines.CancellationException) {
                throw cancelled
            } catch (error: Exception) {
                onError(error)
            } finally {
                sentence.rate.complete(24000)
                sentence.pieces.close()
            }
        }
    }

    private suspend fun play(current: Reply) {
        var framesWritten = 0L
        try {
            try {
                for (sentence in current.sentences) {
                    val rate = sentence.rate.await()
                    val audio = track?.takeIf { it.sampleRate == rate } ?: newTrack(rate).also { release(); track = it; framesWritten = 0; resetLoudness(rate); it.play() }
                    var first = true
                    var waiting = ByteArray(0)
                    for (piece in sentence.pieces) {
                        if (first) {
                            // Wait until a little is buffered so the start does not stutter.
                            waiting += piece
                            if (waiting.size / 2.0 / rate < prebufferSeconds) continue
                            first = false
                            framesWritten += write(audio, waiting)
                            waiting = ByteArray(0)
                        } else framesWritten += write(audio, piece)
                    }
                    if (waiting.isNotEmpty()) framesWritten += write(audio, waiting)
                }
                // Everything is queued: wait for the sound to finish, but never forever.
                val audio = track
                var waited = 0
                while (audio != null && current.scope.isActive && audio.playbackHeadPosition < framesWritten && waited < 600) { delay(50); waited += 1 }
            } catch (released: IllegalStateException) {
                // Any other AudioTrack call (play(), sampleRate, playbackHeadPosition) can throw the same way as
                // write() if stop()/release() races it from another thread. Treat it like a clean end of playback.
            }
        } finally {
            if (reply === current) { reply = null; release(); setActive(false) }
        }
    }

    private fun write(audio: AudioTrack, bytes: ByteArray): Long =
        writeBytes(bytes) { offset, length -> audio.write(bytes, offset, length) }

    /**
     * Writes [bytes] in a loop via [writeFn] (bound to one [AudioTrack] by the caller), measuring loudness as it goes.
     * [AudioTrack.write] throws [IllegalStateException] when the track was paused/flushed/released concurrently on
     * another thread - which happens routinely here: [stop] (called from [leave]/[onCleared] when the user backs out
     * of the chat, or from a new [begin] interrupting an old reply) races with a [play] coroutine still mid-sentence.
     * That race is expected, not a bug, so it must not escape as an uncaught exception and crash the whole app.
     */
    internal fun writeBytes(bytes: ByteArray, writeFn: (offset: Int, length: Int) -> Int): Long {
        measure(bytes)
        var offset = 0
        try {
            while (offset < bytes.size) {
                val count = writeFn(offset, bytes.size - offset)
                if (count <= 0) break
                offset += count
            }
        } catch (released: IllegalStateException) {
            // Track was stopped/released on another thread while this write was in flight. Stop writing; the
            // caller treats a short count the same as a clean end of this sentence.
        }
        return offset / 2L
    }

    private fun newTrack(rate: Int): AudioTrack {
        val minimum = AudioTrack.getMinBufferSize(rate, AudioFormat.CHANNEL_OUT_MONO, AudioFormat.ENCODING_PCM_16BIT)
        return AudioTrack.Builder()
            .setAudioAttributes(AudioAttributes.Builder().setUsage(AudioAttributes.USAGE_MEDIA).setContentType(AudioAttributes.CONTENT_TYPE_SPEECH).build())
            .setAudioFormat(AudioFormat.Builder().setSampleRate(rate).setEncoding(AudioFormat.ENCODING_PCM_16BIT).setChannelMask(AudioFormat.CHANNEL_OUT_MONO).build())
            .setBufferSizeInBytes(maxOf(minimum, rate))
            .setTransferMode(AudioTrack.MODE_STREAM)
            .build()
    }

    private fun release() {
        val old = track ?: return
        track = null
        runCatching { old.pause(); old.flush() }
        runCatching { old.release() }
    }
}
