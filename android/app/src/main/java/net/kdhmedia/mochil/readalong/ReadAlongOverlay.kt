package net.kdhmedia.mochil.readalong

import net.kdhmedia.mochil.narration.NarrationPlan
import org.json.JSONObject

/** One stretch of a recorded audio file that speaks one element of one chapter. [audio] indexes [ReadAlongOverlay.audio]. */
data class ReadAlongClip(val href: String, val id: String, val audio: Int, val begin: Double, val end: Double)

/** The narration of a read-along EPUB (EPUB 3 Media Overlays, as Storyteller makes them), in reading order. */
class ReadAlongOverlay(val audio: List<String>, val clips: List<ReadAlongClip>) {
    /** The clips of one chapter, as indexes, in reading order. */
    fun clipsOf(href: String): List<Int> = clips.indices.filter { sameChapter(clips[it].href, href) }

    /**
     * Where to start reading from [href]: that chapter's clip at [progression] of the way through, or, for a
     * chapter with no narration (a cover, a title page), the start of the next chapter that has some.
     */
    fun startClipFrom(order: List<String>, href: String, progression: Double): Int {
        val here = startClip(href, progression)
        if (here >= 0) return here
        val at = order.indexOfFirst { sameChapter(it, href) }
        if (at < 0) return 0
        order.drop(at + 1).forEach { next -> clipsOf(next).firstOrNull()?.let { return it } }
        return 0
    }

    /** The clip to start from for a page: the chapter's clip at that share of the way through it. */
    fun startClip(href: String, progression: Double): Int {
        val own = clips.indices.filter { sameChapter(clips[it].href, href) }
        if (own.isEmpty()) return -1
        val share = progression.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0) ?: 0.0
        return own[(share * own.size).toInt().coerceAtMost(own.lastIndex)]
    }

    /** The clip speaking at [seconds] into audio file [file], or -1 in a gap. [hint] is tried first, as time mostly moves forward. */
    fun clipAt(file: Int, seconds: Double, hint: Int = 0): Int {
        fun fits(i: Int) = clips.getOrNull(i)?.let { it.audio == file && seconds >= it.begin - 0.02 && seconds < it.end } == true
        if (fits(hint)) return hint
        if (fits(hint + 1)) return hint + 1
        return clips.indices.firstOrNull(::fits) ?: -1
    }

    /** Where within its chapter a clip is, 0 to 1, for remembering the reading position. */
    fun progressionOf(index: Int): Double {
        val clip = clips.getOrNull(index) ?: return 0.0
        val own = clips.indices.filter { sameChapter(clips[it].href, clip.href) }
        return if (own.size < 2) 0.0 else own.indexOf(index).coerceAtLeast(0).toDouble() / own.size
    }

    private fun sameChapter(a: String, b: String) = NarrationPlan.normalizedHref(a) == NarrationPlan.normalizedHref(b)

    companion object {
        /** A clip this long is a stretch the aligner could not split into sentences: it is played, but not highlighted. */
        const val MAX_SENTENCE_SECONDS = 90.0

        /** Null when the book has no narration. */
        fun parse(json: JSONObject): ReadAlongOverlay? {
            val files = json.optJSONArray("audio")?.let { array -> (0 until array.length()).map { array.getString(it) } }.orEmpty()
            val chapters = json.optJSONArray("chapters") ?: return null
            val clips = ArrayList<ReadAlongClip>()
            for (c in 0 until chapters.length()) {
                val chapter = chapters.getJSONObject(c)
                val list = chapter.optJSONArray("clips") ?: continue
                for (i in 0 until list.length()) {
                    val clip = list.getJSONArray(i)
                    val audio = clip.getInt(1)
                    val begin = clip.getDouble(2)
                    val end = clip.getDouble(3)
                    if (end > begin && audio in files.indices) clips += ReadAlongClip(chapter.getString("href"), clip.getString(0), audio, begin, end)
                }
            }
            return if (clips.isEmpty()) null else ReadAlongOverlay(files, clips)
        }
    }
}
