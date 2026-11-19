package net.kdhmedia.mochil.hermes

import android.annotation.SuppressLint
import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import android.media.audiofx.AcousticEchoCanceler
import android.media.audiofx.NoiseSuppressor
import android.os.Handler
import android.os.Looper
import kotlin.concurrent.thread
import kotlin.math.sqrt

/**
 * Listens through the microphone and reports whole utterances as small WAVs. Nothing leaves the phone until an utterance
 * is finished, and then it goes only to your own server. The source is the "voice communication" one, which turns on the
 * phone's echo cancellation so her voice from the speaker is less likely to be taken for yours.
 */
class HermesMicrophone(private val handlers: TalkHandlers) : TalkMicrophone {
    private val main = Handler(Looper.getMainLooper())
    @Volatile private var vad = HermesVad()
    private val captureRunGate = CaptureRunGate()
    @Volatile private var running = false
    @Volatile private var muted = false
    @Volatile private var pickier = false
    private var worker: Thread? = null
    private var record: AudioRecord? = null
    private var effects: List<android.media.audiofx.AudioEffect> = emptyList()

    @Synchronized
    @SuppressLint("MissingPermission") // The screen asks for the permission before it starts the microphone.
    override fun start() {
        if (running) return
        val run = captureRunGate.start()
        val minimum = AudioRecord.getMinBufferSize(RATE, AudioFormat.CHANNEL_IN_MONO, AudioFormat.ENCODING_PCM_16BIT)
        val audio = AudioRecord(MediaRecorder.AudioSource.VOICE_COMMUNICATION, RATE, AudioFormat.CHANNEL_IN_MONO, AudioFormat.ENCODING_PCM_16BIT, maxOf(minimum, BLOCK * 8))
        if (audio.state != AudioRecord.STATE_INITIALIZED) { audio.release(); throw IllegalStateException("The microphone could not be opened") }
        effects = listOfNotNull(
            if (AcousticEchoCanceler.isAvailable()) AcousticEchoCanceler.create(audio.audioSessionId)?.also { it.enabled = true } else null,
            if (NoiseSuppressor.isAvailable()) NoiseSuppressor.create(audio.audioSessionId)?.also { it.enabled = true } else null,
        )
        record = audio
        val runVad = HermesVad()
        vad = runVad
        running = true
        startCaptureOrCleanup(start = { audio.startRecording() }) {
            captureRunGate.stop()
            running = false
            effects.forEach { runCatching { it.release() } }
            effects = emptyList()
            record = null
            runCatching { audio.release() }
        }
        worker = thread(name = "hermes-microphone", isDaemon = true) {
            // A blocked old read can finish after stop() and a fast restart; its error belongs only to that capture run.
            try { loop(audio, run, runVad) } catch (error: Throwable) {
                if (captureRunGate.isCurrent(run)) main.post {
                    if (captureRunGate.isCurrent(run)) handlers.onError(error)
                }
            }
        }
    }

    private fun loop(audio: AudioRecord, run: Int, vad: HermesVad) {
        val block = ShortArray(BLOCK)
        var preroll = ArrayDeque<ShortArray>()
        var utterance: MutableList<ShortArray>? = null
        while (captureRunGate.isCurrent(run)) {
            var read = 0
            while (read < BLOCK && captureRunGate.isCurrent(run)) {
                val count = audio.read(block, read, BLOCK - read)
                if (count == AudioRecord.ERROR_DEAD_OBJECT) throw IllegalStateException("The microphone was taken by another app")
                if (count <= 0) { if (!captureRunGate.isCurrent(run)) return; Thread.sleep(5); continue }
                read += count
            }
            if (!captureRunGate.isCurrent(run)) return
            if (muted) { vad.reset(); utterance = null; preroll.clear(); continue }
            var sum = 0.0
            for (sample in block) { val v = sample / 32768.0; sum += v * v }
            val event = vad.feed(sqrt(sum / BLOCK), BLOCK_MS)
            if (!captureRunGate.isCurrent(run)) return
            val copy = block.copyOf()
            if (utterance != null) utterance.add(copy) else { preroll.addLast(copy); if (preroll.size > PREROLL) preroll.removeFirst() }
            when {
                event == HermesVad.Event.Start -> {
                    utterance = preroll.toMutableList(); preroll = ArrayDeque()
                    main.post { if (captureRunGate.isCurrent(run)) handlers.onSpeechStart() }
                }
                event == HermesVad.Event.End && utterance != null -> {
                    // Keep only a little of the closing silence.
                    val keep = maxOf(1, utterance.size - (SILENCE_TRIM_MS / BLOCK_MS).toInt())
                    val blocks = utterance.take(keep + 10)
                    utterance = null
                    val pcm = ShortArray(blocks.size * BLOCK)
                    blocks.forEachIndexed { i, b -> b.copyInto(pcm, i * BLOCK) }
                    val wav = encodeWav16k(pcm)
                    val ms = (blocks.size * BLOCK_MS).toInt()
                    main.post { if (captureRunGate.isCurrent(run)) handlers.onUtterance(wav, ms) }
                }
            }
        }
    }

    @Synchronized
    override fun stop() {
        captureRunGate.stop()
        running = false
        effects.forEach { runCatching { it.release() } }
        effects = emptyList()
        record?.let { runCatching { it.stop() }; runCatching { it.release() } }
        record = null
        worker = null
    }

    override fun setMuted(muted: Boolean) { this.muted = muted }

    override fun hearAbove(speaking: Boolean) {
        if (pickier == speaking) return
        pickier = speaking
        if (speaking) vad.sensitivity(factor = 5.0, minSpeech = 420.0) else vad.sensitivity(factor = 3.0, minSpeech = 240.0)
    }

    private companion object {
        const val RATE = 16000
        const val BLOCK = 320            // 20 ms
        const val BLOCK_MS = 20.0
        const val PREROLL = 28           // about 560 ms kept from before speech was confirmed
        const val SILENCE_TRIM_MS = 700.0
    }
}
