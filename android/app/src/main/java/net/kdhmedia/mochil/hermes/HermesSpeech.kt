package net.kdhmedia.mochil.hermes

// Pure speech helpers, the same rules as the website (hermes-voice.js, hermes-listen.js, hermes-talk.js), so the phone
// and the browser cut sentences, hear you and take turns the same way.

private val EMOJI = Regex("[\\p{So}\\x{1F300}-\\x{1FAFF}\\x{2600}-\\x{27BF}\\u200D\\uFE0F]")

/** What is worth hearing: no code, links, markdown marks, emoji, or stage directions such as *smiles softly*. */
fun speechText(raw: String): String {
    var text = raw
    text = Regex("```[\\s\\S]*?```").replace(text, " ")
    text = Regex("`([^`]*)`").replace(text, "\$1")
    text = Regex("\\[([^\\]]*)\\]\\([^)]*\\)").replace(text, "\$1")
    text = Regex("https?://\\S+").replace(text, "")
    text = Regex("\\*\\*([^*]+)\\*\\*").replace(text, "\$1")
    text = Regex("\\*([^*]+)\\*").replace(text) { match ->
        val inner = match.groupValues[1]
        if (inner.trim().split(Regex("\\s+")).size >= 2) " " else inner
    }
    text = Regex("^\\s{0,3}#{1,6}\\s+", RegexOption.MULTILINE).replace(text, "")
    text = Regex("^\\s*[-*•]\\s+", RegexOption.MULTILINE).replace(text, "")
    text = Regex("^\\s*\\d+[.)]\\s+", RegexOption.MULTILINE).replace(text, "")
    text = EMOJI.replace(text, "")
    text = Regex("[_~>|]").replace(text, " ")
    return Regex("\\s+").replace(text, " ").trim()
}

data class TakenSentences(val sentences: List<String>, val rest: String)

private val BOUNDARY = Regex("[.!?…]+[\"'”)\\]]*\\s+|\\n+")

/**
 * Takes complete sentences off the front of a growing text. A sentence is complete once its end mark is followed by a
 * space or a new line; short ones wait and join the next, so nothing tiny is spoken alone. With [final] the rest is flushed.
 */
fun takeSentences(buffer: String, final: Boolean = false, minimum: Int = 24): TakenSentences {
    val sentences = mutableListOf<String>()
    var current = ""
    var last = 0
    for (match in BOUNDARY.findAll(buffer)) {
        val end = match.range.last + 1
        current += buffer.substring(last, end)
        last = end
        if (speechText(current).length >= minimum) { sentences += current; current = "" }
    }
    var rest = current + buffer.substring(last)
    if (final) {
        if (speechText(rest).isNotEmpty()) sentences += rest
        rest = ""
    }
    return TakenSentences(sentences.map(::speechText).filter { it.isNotEmpty() }, rest)
}

/**
 * Voice activity detection on 20 ms loudness readings. Speech starts when it stays above the room's noise for a moment
 * and ends after a short silence, so there is no long fixed wait. The noise floor follows the room while nobody talks.
 */
class HermesVad(
    private var minSpeechMs: Double = 240.0,
    private val silenceMs: Double = 800.0,
    private val maxMs: Double = 55000.0,
    private var factor: Double = 3.0,
    private val floorMin: Double = 0.006,
    private val threshold: Double = 0.02,
) {
    enum class Event { Start, End }

    private var floor = floorMin
    private var loud = 0.0
    private var speaking = false
    private var speechMs = 0.0
    private var quiet = 0.0

    private fun limit() = maxOf(threshold, floor * factor)

    val isSpeaking: Boolean get() = speaking

    fun sensitivity(factor: Double? = null, minSpeech: Double? = null) {
        factor?.let { this.factor = it }
        minSpeech?.let { minSpeechMs = it }
    }

    fun reset() { loud = 0.0; speaking = false; speechMs = 0.0; quiet = 0.0 }

    /** One loudness reading; returns Start or End when the state changes. */
    fun feed(rms: Double, dt: Double): Event? {
        if (!speaking) {
            if (rms > limit()) loud += dt
            else {
                loud = 0.0
                floor = minOf(0.05, maxOf(floorMin, floor * 0.95 + rms * 0.05))
            }
            if (loud >= minSpeechMs) { speaking = true; speechMs = loud; quiet = 0.0; return Event.Start }
            return null
        }
        speechMs += dt
        if (rms > limit() * 0.6) quiet = 0.0 else quiet += dt
        if (quiet >= silenceMs || speechMs >= maxMs) { speaking = false; loud = 0.0; return Event.End }
        return null
    }
}

enum class HeardKind { Empty, Stop, Phantom, Say }

private val PHANTOM = setOf("thanks for watching", "thank you for watching", "please subscribe", "subscribe", "you", "the end", "blank audio", "silence")
private val STOP = setOf("stop", "stop listening", "stop talking", "stop the conversation", "end conversation")

fun normalizeHeard(text: String): String =
    text.lowercase().replace(Regex("\\[[^\\]]*\\]"), " ").replace(Regex("[^\\p{L}\\p{N}' ]"), " ").replace(Regex("\\s+"), " ").trim()

/** What to do with what was heard: nothing (silence or a Whisper phantom), end the conversation, or say it to her. */
fun classifyHeard(text: String, ms: Int = 0): HeardKind {
    val heard = normalizeHeard(text)
    return when {
        heard.isEmpty() -> HeardKind.Empty
        heard in STOP -> HeardKind.Stop
        heard in PHANTOM && ms < 2500 -> HeardKind.Phantom
        else -> HeardKind.Say
    }
}

/** A 16 kHz mono 16-bit WAV, the format the server's speech recognition takes. */
fun encodeWav16k(pcm: ShortArray): ByteArray {
    val data = pcm.size * 2
    val out = java.nio.ByteBuffer.allocate(44 + data).order(java.nio.ByteOrder.LITTLE_ENDIAN)
    out.put("RIFF".toByteArray()).putInt(36 + data).put("WAVE".toByteArray()).put("fmt ".toByteArray())
    out.putInt(16).putShort(1).putShort(1).putInt(16000).putInt(32000).putShort(2).putShort(16)
    out.put("data".toByteArray()).putInt(data)
    for (sample in pcm) out.putShort(sample)
    return out.array()
}

/** Joins whole 16-bit samples from pieces of bytes that may split a sample in two. */
class SampleSplitter {
    private var carry: Byte? = null

    fun feed(bytes: ByteArray, length: Int = bytes.size): ByteArray {
        var data = bytes.copyOf(length)
        carry?.let { data = byteArrayOf(it) + data; carry = null }
        if (data.size % 2 == 1) { carry = data.last(); data = data.copyOf(data.size - 1) }
        return data
    }
}
