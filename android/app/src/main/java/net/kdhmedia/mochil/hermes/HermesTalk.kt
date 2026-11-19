package net.kdhmedia.mochil.hermes

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

enum class TalkMode { Off, Listening, Hearing, Transcribing, Waiting }

data class TalkOptions(val handsFree: Boolean, val bargeIn: Boolean)

/** What the microphone side offers the conversation: start, stop, ignore the mic, and be pickier while she speaks. */
interface TalkMicrophone {
    fun start()
    fun stop()
    fun setMuted(muted: Boolean)
    fun hearAbove(speaking: Boolean)
}

interface TalkHandlers {
    fun onSpeechStart()
    fun onUtterance(wav: ByteArray, ms: Int)
    fun onError(error: Throwable)
}

/**
 * The voice conversation: listen, understand, send, wait for the answer, listen again. The decisions live here so they
 * can be tested; the screen supplies the microphone, the transcription call, sending and interrupting.
 */
class HermesTalk(
    private val scope: CoroutineScope,
    private val makeMicrophone: (TalkHandlers) -> TalkMicrophone,
    private val isSpeaking: () -> Boolean,
    private val transcribe: suspend (ByteArray) -> String,
    private val send: (text: String, interrupted: Boolean) -> Unit,
    private val interrupt: () -> Unit,
    private val options: () -> TalkOptions,
    private val onMode: (TalkMode) -> Unit = {},
    private val onError: (Throwable) -> Unit = {},
) {
    var mode: TalkMode = TalkMode.Off
        private set
    private var microphone: TalkMicrophone? = null
    private var interrupted = false

    private fun set(next: TalkMode) { mode = next; refresh(); onMode(next) }

    /** Muted while she answers unless you may talk over her; pickier while her voice is playing. */
    fun refresh() {
        val mic = microphone ?: return
        val speaking = isSpeaking()
        val busy = mode == TalkMode.Transcribing || mode == TalkMode.Waiting || speaking
        mic.setMuted(mode == TalkMode.Transcribing || (busy && !options().bargeIn))
        mic.hearAbove(speaking)
    }

    private fun heard(wav: ByteArray, ms: Int) {
        set(TalkMode.Transcribing)
        if (!options().handsFree) { microphone?.stop(); microphone = null }
        scope.launch {
            val text = try { transcribe(wav) } catch (error: Throwable) { onError(error); return@launch back() }
            when (classifyHeard(text, ms)) {
                HeardKind.Stop -> stop()
                HeardKind.Empty, HeardKind.Phantom -> back()
                HeardKind.Say -> {
                    set(TalkMode.Waiting)
                    val flag = interrupted
                    interrupted = false
                    send(text, flag)
                }
            }
        }
    }

    /** Back to listening, or to rest when each turn needs a tap. */
    private fun back() { if (options().handsFree && microphone != null) set(TalkMode.Listening) else set(TalkMode.Off) }

    private val handlers = object : TalkHandlers {
        override fun onSpeechStart() {
            if (isSpeaking() || mode == TalkMode.Waiting) { interrupted = true; interrupt() }
            set(TalkMode.Hearing)
        }
        override fun onUtterance(wav: ByteArray, ms: Int) = heard(wav, ms)
        override fun onError(error: Throwable) { this@HermesTalk.onError(error); stop() } // the microphone itself failed
    }

    fun toggle() {
        if (mode != TalkMode.Off) return stop()
        val mic = makeMicrophone(handlers)
        microphone = mic
        try { mic.start() } catch (error: Throwable) { microphone = null; onError(error); return }
        set(TalkMode.Listening)
    }

    fun stop() {
        microphone?.stop()
        microphone = null
        interrupted = false
        set(TalkMode.Off)
    }

    /** The reply (and its text) is complete; the conversation goes on when it is hands-free. */
    fun replyDone() { if (mode == TalkMode.Waiting) back() else refresh() }
}
