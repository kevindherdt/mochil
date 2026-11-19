package net.kdhmedia.mochil.reader.epub

import android.content.Context
import java.time.LocalDate
import org.json.JSONArray
import org.json.JSONObject

data class ReadingStatistics(
    val totalReadingMillis: Long = 0,
    val todayReadingMillis: Long = 0,
    val positionChanges: Int = 0,
    val averagePositionMillis: Long? = null,
    val bookRemainingMillis: Long? = null,
    val chapterRemainingMillis: Long? = null,
)

internal data class ReadingStatisticsState(
    val totalReadingMillis: Long = 0,
    val todayKey: String = "",
    val todayReadingMillis: Long = 0,
    val positionChanges: Int = 0,
    val positionWindow: List<Double> = emptyList(),
    val bookWindow: List<Double> = emptyList(),
    val chapterWindow: List<Double> = emptyList(),
    val lastAtMillis: Long = 0,
    val lastTotalProgression: Double? = null,
    val lastChapterProgression: Double? = null,
    val lastHref: String = "",
)

internal const val MIN_PACE_SAMPLES = 3
internal const val MAX_PACE_SAMPLES = 15
internal const val PACE_OUTLIER_FACTOR = 4.0
internal const val MIN_SECONDS_PER_BOOK = 300.0
internal const val MAX_SECONDS_PER_BOOK = 60_000.0

/** The middle of the recent samples: one burst of fast turns moves a median far less than it moves an average. */
internal fun medianPace(values: List<Double>): Double? {
    val sorted = values.filter { it.isFinite() && it > 0 }.sorted()
    if (sorted.isEmpty()) return null
    val middle = sorted.size / 2
    return if (sorted.size % 2 == 1) sorted[middle] else (sorted[middle - 1] + sorted[middle]) / 2
}

/**
 * Seconds this interval implies for a whole book (or chapter), or null when it cannot be a reading pace.
 *
 * Two gates. An absolute one: nobody reads a book in five minutes, or spends sixteen hours on one turn. Then, once a
 * pace is known, a relative one: an interval more than [PACE_OUTLIER_FACTOR] times faster or slower than the usual
 * pace is a skim, a jump to the index or a distraction. It is real elapsed time, but it says nothing about speed.
 */
internal fun paceSample(elapsedMillis: Long, progressDelta: Double?, window: List<Double>): Double? {
    if (progressDelta == null || progressDelta <= 0 || elapsedMillis <= 0) return null
    val sample = elapsedMillis / 1_000.0 / progressDelta
    if (!sample.isFinite() || sample < MIN_SECONDS_PER_BOOK || sample > MAX_SECONDS_PER_BOOK) return null
    val typical = medianPace(window) ?: return sample
    if (window.size >= MIN_PACE_SAMPLES && (sample < typical / PACE_OUTLIER_FACTOR || sample > typical * PACE_OUTLIER_FACTOR)) return null
    return sample
}

internal fun withPaceSample(window: List<Double>, sample: Double?): List<Double> =
    if (sample == null) window else (window + sample).takeLast(MAX_PACE_SAMPLES)

/** How much of the book (or chapter) is left at this pace, or null while the pace is still unknown. */
internal fun remainingMillis(window: List<Double>, progression: Double?): Long? {
    if (window.size < MIN_PACE_SAMPLES || progression == null) return null
    val pace = medianPace(window) ?: return null
    return (pace * (1.0 - progression.coerceIn(0.0, 1.0)) * 1_000).toLong().coerceAtLeast(0)
}

class ReadingStatisticsStore(context: Context) {
    private val preferences = context.getSharedPreferences("epub_reading_statistics", Context.MODE_PRIVATE)

    fun record(publicationKey: String, totalProgression: Double?, chapterProgression: Double?, href: String, nowMillis: Long = System.currentTimeMillis()): ReadingStatistics {
        val today = LocalDate.now().toString()
        val previous = decodeStatistics(preferences.getString(publicationKey, null)).let {
            if (it.todayKey == today) it else it.copy(todayKey = today, todayReadingMillis = 0)
        }
        val elapsed = nowMillis - previous.lastAtMillis
        val activeElapsed = elapsed.takeIf { previous.lastAtMillis > 0 && it in MIN_ACTIVE_INTERVAL_MS..MAX_ACTIVE_INTERVAL_MS } ?: 0
        val moved = positionChanged(previous, totalProgression, chapterProgression, href)
        val totalDelta = positiveDelta(previous.lastTotalProgression, totalProgression)
        val chapterDelta = if (previous.lastHref == href) positiveDelta(previous.lastChapterProgression, chapterProgression) else null
        val turned = moved && activeElapsed > 0
        val updated = previous.copy(
            totalReadingMillis = previous.totalReadingMillis + activeElapsed,
            todayReadingMillis = previous.todayReadingMillis + activeElapsed,
            positionChanges = previous.positionChanges + if (turned) 1 else 0,
            positionWindow = if (turned) (previous.positionWindow + activeElapsed.toDouble()).takeLast(MAX_PACE_SAMPLES) else previous.positionWindow,
            bookWindow = withPaceSample(previous.bookWindow, if (turned) paceSample(activeElapsed, totalDelta, previous.bookWindow) else null),
            chapterWindow = withPaceSample(previous.chapterWindow, if (turned) paceSample(activeElapsed, chapterDelta, previous.chapterWindow) else null),
            lastAtMillis = nowMillis,
            lastTotalProgression = totalProgression?.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0),
            lastChapterProgression = chapterProgression?.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0),
            lastHref = href,
        )
        preferences.edit().putString(publicationKey, encodeStatistics(updated).toString()).apply()
        return updated.snapshot()
    }

    fun snapshot(publicationKey: String): ReadingStatistics = decodeStatistics(preferences.getString(publicationKey, null)).snapshot()

    private fun ReadingStatisticsState.snapshot(): ReadingStatistics = ReadingStatistics(
        totalReadingMillis = totalReadingMillis,
        todayReadingMillis = if (todayKey == LocalDate.now().toString()) todayReadingMillis else 0,
        positionChanges = positionChanges,
        averagePositionMillis = medianPace(positionWindow)?.takeIf { positionWindow.size >= MIN_PACE_SAMPLES }?.toLong(),
        bookRemainingMillis = remainingMillis(bookWindow, lastTotalProgression),
        chapterRemainingMillis = remainingMillis(chapterWindow, lastChapterProgression),
    )

    private fun positionChanged(previous: ReadingStatisticsState, total: Double?, chapter: Double?, href: String): Boolean =
        previous.lastAtMillis > 0 && (previous.lastHref != href || previous.lastTotalProgression != total || previous.lastChapterProgression != chapter)

    private fun positiveDelta(previous: Double?, current: Double?): Double? {
        if (previous == null || current == null || !current.isFinite()) return null
        return (current - previous).takeIf { it > MIN_PROGRESS_DELTA && it <= MAX_PROGRESS_DELTA }
    }

    private companion object {
        const val MIN_ACTIVE_INTERVAL_MS = 500L
        const val MAX_ACTIVE_INTERVAL_MS = 120_000L
        const val MIN_PROGRESS_DELTA = 0.000001
        const val MAX_PROGRESS_DELTA = 0.25
    }
}

internal fun encodeStatistics(value: ReadingStatisticsState): JSONObject = JSONObject().apply {
    put("total", value.totalReadingMillis); put("today_key", value.todayKey); put("today", value.todayReadingMillis)
    put("turns", value.positionChanges)
    put("position_window", JSONArray(value.positionWindow))
    put("book_window", JSONArray(value.bookWindow))
    put("chapter_window", JSONArray(value.chapterWindow))
    put("last_at", value.lastAtMillis); put("last_total", value.lastTotalProgression); put("last_chapter", value.lastChapterProgression); put("last_href", value.lastHref)
}

internal fun decodeStatistics(encoded: String?): ReadingStatisticsState = runCatching {
    val json = JSONObject(encoded ?: return@runCatching ReadingStatisticsState())
    ReadingStatisticsState(
        totalReadingMillis = json.optLong("total"),
        todayKey = json.optString("today_key"),
        todayReadingMillis = json.optLong("today"),
        positionChanges = json.optInt("turns"),
        // An install that predates the windows stored one running average per measure. Seed the window with it so a
        // pace that is already known keeps working instead of starting over.
        positionWindow = statisticsWindow(json, "position_window").ifEmpty { legacyPace(json.optDouble("average", 0.0)) },
        bookWindow = statisticsWindow(json, "book_window").ifEmpty { legacyPace(json.optDouble("book_rate", 0.0)) },
        chapterWindow = statisticsWindow(json, "chapter_window").ifEmpty { legacyPace(json.optDouble("chapter_rate", 0.0)) },
        lastAtMillis = json.optLong("last_at"),
        lastTotalProgression = json.optDouble("last_total").takeIf { json.has("last_total") && it.isFinite() },
        lastChapterProgression = json.optDouble("last_chapter").takeIf { json.has("last_chapter") && it.isFinite() },
        lastHref = json.optString("last_href"),
    )
}.getOrDefault(ReadingStatisticsState())

private fun statisticsWindow(json: JSONObject, key: String): List<Double> {
    val array = json.optJSONArray(key) ?: return emptyList()
    return (0 until array.length()).mapNotNull { index -> array.optDouble(index).takeIf { it.isFinite() && it > 0 } }
        .takeLast(MAX_PACE_SAMPLES)
}

private fun legacyPace(value: Double): List<Double> =
    if (value.isFinite() && value > 0) List(MIN_PACE_SAMPLES) { value } else emptyList()

internal fun readingDurationLabel(millis: Long?): String? {
    if (millis == null || millis <= 0) return null
    val minutes = ((millis + 30_000) / 60_000).coerceAtLeast(1)
    return if (minutes < 60) "$minutes min" else "${minutes / 60} hr ${minutes % 60} min"
}

internal fun positionDurationLabel(millis: Long?): String? {
    if (millis == null || millis <= 0) return null
    return if (millis < 60_000) "${((millis + 500) / 1_000).coerceAtLeast(1)} sec" else readingDurationLabel(millis)
}

/** "About 1.4 pages a minute", from how long one page (or position) takes. */
internal fun readingSpeedLabel(averagePositionMillis: Long?): String? {
    val average = averagePositionMillis?.takeIf { it > 0 } ?: return null
    val perMinute = 60_000.0 / average
    return when {
        perMinute >= 10 -> "About ${(perMinute + 0.5).toInt()} pages a minute"
        perMinute >= 1 -> "About ${(perMinute * 10).toInt() / 10.0} pages a minute"
        else -> "About 1 page every ${readingDurationLabel(average) ?: "${((average + 500) / 1_000).coerceAtLeast(1)} sec"}"
    }
}
