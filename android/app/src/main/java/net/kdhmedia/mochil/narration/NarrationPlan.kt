package net.kdhmedia.mochil.narration

import java.net.URLDecoder

/** The decisions about what to read and in which order, apart from any player or page. */
object NarrationPlan {
    /** A chapter's address with fragment, leading slash and escapes removed, for comparing. */
    fun normalizedHref(href: String): String =
        runCatching { URLDecoder.decode(href.substringBefore('#').trimStart('/'), "UTF-8") }.getOrDefault(href).trimStart('/')

    /** The server's chapter that is the reader's current resource, or null when it is not one. */
    fun chapterFor(href: String, chapters: List<NarrationChapter>): NarrationChapter? {
        val wanted = normalizedHref(href)
        return chapters.firstOrNull { normalizedHref(it.href) == wanted }
    }

    /** The first sentence at or after a position, so reading starts where the page starts. */
    fun startSentence(sentences: List<NarrationSentence>, progression: Double): Int {
        if (sentences.isEmpty()) return 0
        val from = progression.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0) ?: 0.0
        val at = sentences.indexOfFirst { it.progression + EPSILON >= from }
        return if (at < 0) sentences.lastIndex else at
    }

    /** The chapter after this one that is worth reading, or null at the end of the book. */
    fun nextChapter(chapters: List<NarrationChapter>, after: Int): NarrationChapter? =
        chapters.firstOrNull { it.index > after && !it.matter }

    /** How many sentences are queued ahead before the next chapter is added to the queue. */
    const val QUEUE_AHEAD = 8

    private const val EPSILON = 1e-6
}

/** When narration pauses by itself. */
sealed interface NarrationSleep {
    data object Off : NarrationSleep
    data class After(val minutes: Int) : NarrationSleep
    data object EndOfChapter : NarrationSleep
}

/** Speeds the voice can keep up with: it speaks only a little faster than it is heard at 1×. */
val recordingSpeeds = listOf(0.75f, 1f, 1.25f, 1.5f, 1.75f, 2f)
val narrationSpeeds = listOf(0.8f, 1f, 1.15f, 1.3f, 1.5f)

/** Volume while the voice fades out before a sleep timer pauses it. */
fun sleepFadeVolume(remainingMs: Long, fadeMs: Long = 10_000): Float =
    if (remainingMs >= fadeMs) 1f else (remainingMs.coerceAtLeast(0) / fadeMs.toFloat())
