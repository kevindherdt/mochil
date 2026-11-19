package net.kdhmedia.mochil.video

import org.json.JSONArray
import org.json.JSONObject

internal data class VideoPlaybackEntry(
    val libraryId: String,
    val itemId: String,
    val title: String,
    val url: String,
    val fallbackUrl: String,
    val positionMs: Long,
    val durationSeconds: Double,
    val subtitles: String,
    val revision: String = "",
    val progressRevision: Long = 0,
    val finished: Boolean = false,
)

internal data class VideoPlaybackQueue(val entries: List<VideoPlaybackEntry>, val currentIndex: Int, val startOverrideMs: Long? = null)

internal fun VideoPlaybackQueue.startPositionMs(): Long = startOverrideMs ?: entries[currentIndex].let {
    if (it.finished) 0L else it.positionMs
}

internal fun videoPlaybackIdentityKey(libraryId: String, itemId: String): String =
    "${libraryId.length}:$libraryId${itemId.length}:$itemId"

internal fun videoPlaybackWindow(items: List<VideoItem>, current: VideoItem, maximumEntries: Int = 50): Pair<List<VideoItem>, Int> {
    val playable = if (current.show.isBlank()) listOf(current) else items.filter { it.playable && it.show == current.show }
        .ifEmpty { listOf(current) }
    val currentIndex = playable.indexOfFirst { it.libraryId == current.libraryId && it.id == current.id }.takeIf { it >= 0 } ?: return listOf(current) to 0
    val boundedMaximum = maximumEntries.coerceAtLeast(1)
    val start = (currentIndex - 5).coerceAtLeast(0).coerceAtMost((playable.size - boundedMaximum).coerceAtLeast(0))
    val window = playable.subList(start, (start + boundedMaximum).coerceAtMost(playable.size))
    return window to (currentIndex - start)
}

internal fun encodeVideoPlaybackQueue(queue: VideoPlaybackQueue): String = JSONObject()
    .put("current", queue.currentIndex)
    .put("entries", JSONArray().apply {
        queue.entries.forEach { entry -> put(JSONObject()
            .put("library", entry.libraryId).put("item", entry.itemId).put("title", entry.title).put("revision", entry.revision)
            .put("progress_revision", entry.progressRevision).put("finished", entry.finished)
            .put("url", entry.url).put("fallback", entry.fallbackUrl).put("position_ms", entry.positionMs)
            .put("duration_seconds", entry.durationSeconds).put("subtitles", entry.subtitles)) }
    }).toString()

internal fun decodeVideoPlaybackQueue(encoded: String, maximumBytes: Int = 256 * 1024, maximumEntries: Int = 50): VideoPlaybackQueue? {
    if (encoded.isBlank() || encoded.toByteArray().size > maximumBytes) return null
    return runCatching {
        val root = JSONObject(encoded)
        val values = root.getJSONArray("entries")
        if (values.length() !in 1..maximumEntries) return null
        val entries = List(values.length()) { index -> values.getJSONObject(index).let { value ->
            VideoPlaybackEntry(
                libraryId = value.getString("library").take(160), itemId = value.getString("item").take(512),
                title = value.optString("title").take(240), revision = value.optString("revision").take(160), url = value.getString("url"), fallbackUrl = value.optString("fallback"),
                positionMs = value.optLong("position_ms").coerceAtLeast(0),
                durationSeconds = value.optDouble("duration_seconds").takeIf(Double::isFinite)?.coerceAtLeast(0.0) ?: 0.0,
                subtitles = value.optString("subtitles").take(32 * 1024),
                progressRevision = value.optLong("progress_revision").coerceAtLeast(0),
                finished = value.optBoolean("finished"),
            ).also { require(it.libraryId.isNotBlank() && it.itemId.isNotBlank() && it.url.isNotBlank()) }
        } }
        VideoPlaybackQueue(entries, root.optInt("current").coerceIn(entries.indices))
    }.getOrNull()
}
