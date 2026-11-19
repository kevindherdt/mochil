@file:android.annotation.SuppressLint("ApplySharedPref") // Bookmark mutations must be durable before returning to playback.
package net.kdhmedia.mochil.playback

import android.content.Context
import java.util.UUID
import net.kdhmedia.mochil.audio.AudioChapter
import net.kdhmedia.mochil.audio.AudioTrack
import org.json.JSONArray
import org.json.JSONObject

data class AudiobookBookmark(
    val id: String,
    val libraryId: String,
    val itemId: String,
    val positionMs: Long,
    val label: String,
    val createdAt: Long,
)

internal fun chapterAt(chapters: List<AudioChapter>, positionMs: Long): AudioChapter? {
    val seconds = positionMs.coerceAtLeast(0) / 1_000.0
    return chapters.lastOrNull { chapter ->
        chapter.startSeconds.isFinite() && chapter.startSeconds <= seconds &&
            (!chapter.endSeconds.isFinite() || chapter.endSeconds <= chapter.startSeconds || seconds < chapter.endSeconds)
    }
}

class AudiobookBookmarkStore(context: Context) {
    private val preferences = context.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)

    @Synchronized
    fun list(track: AudioTrack): List<AudiobookBookmark> = decode()
        .filter { it.libraryId == track.libraryId && it.itemId == track.id }
        .sortedBy(AudiobookBookmark::positionMs)

    @Synchronized
    fun add(track: AudioTrack, positionMs: Long): List<AudiobookBookmark> {
        if (track.libraryKind != "audiobooks") return emptyList()
        val boundedPosition = positionMs.coerceIn(0, (track.durationSeconds * 1_000).toLong().coerceAtLeast(0))
        val existing = decode()
        val nearby = existing.firstOrNull { it.libraryId == track.libraryId && it.itemId == track.id && kotlin.math.abs(it.positionMs - boundedPosition) < 1_000 }
        if (nearby == null) {
            val chapter = chapterAt(track.chapters, boundedPosition)
            val bookmark = AudiobookBookmark(UUID.randomUUID().toString(), track.libraryId, track.id, boundedPosition, chapter?.title?.take(120).orEmpty(), System.currentTimeMillis())
            save((existing + bookmark).takeLast(MAX_BOOKMARKS))
        }
        return list(track)
    }

    @Synchronized
    fun remove(track: AudioTrack, bookmarkId: String): List<AudiobookBookmark> {
        save(decode().filterNot { it.id == bookmarkId && it.libraryId == track.libraryId && it.itemId == track.id })
        return list(track)
    }

    private fun decode(): List<AudiobookBookmark> = runCatching {
        val values = JSONArray(preferences.getString(KEY_ITEMS, "[]"))
        val count = minOf(values.length(), MAX_BOOKMARKS)
        List(count) { index -> values.getJSONObject(index).run {
            AudiobookBookmark(
                id = getString("id").take(80), libraryId = getString("library_id").take(160), itemId = getString("item_id").take(160),
                positionMs = optLong("position_ms").coerceAtLeast(0), label = optString("label").take(120), createdAt = optLong("created_at").coerceAtLeast(0),
            )
        } }
    }.getOrDefault(emptyList())

    private fun save(bookmarks: List<AudiobookBookmark>) {
        val values = JSONArray()
        bookmarks.takeLast(MAX_BOOKMARKS).forEach { bookmark -> values.put(JSONObject().apply {
            put("id", bookmark.id); put("library_id", bookmark.libraryId); put("item_id", bookmark.itemId)
            put("position_ms", bookmark.positionMs); put("label", bookmark.label); put("created_at", bookmark.createdAt)
        }) }
        preferences.edit().putString(KEY_ITEMS, values.toString()).commit()
    }

    companion object {
        private const val PREFERENCES = "audiobook_bookmarks"
        private const val KEY_ITEMS = "items"
        private const val MAX_BOOKMARKS = 500
    }
}
