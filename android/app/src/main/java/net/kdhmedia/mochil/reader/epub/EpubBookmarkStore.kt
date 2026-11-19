package net.kdhmedia.mochil.reader.epub

import android.content.Context
import org.json.JSONArray
import org.json.JSONObject
import org.readium.r2.shared.publication.Locator

data class EpubBookmark(val locator: Locator, val createdAt: Long)

class EpubBookmarkStore(context: Context) {
    private val preferences = context.getSharedPreferences("epub_reader_bookmarks", Context.MODE_PRIVATE)

    fun list(publicationKey: String): List<EpubBookmark> = parseBookmarks(preferences.getString(publicationKey, "[]").orEmpty())

    fun toggle(publicationKey: String, locator: Locator): List<EpubBookmark> {
        val existing = list(publicationKey)
        val identifier = locator.bookmarkIdentifier()
        val updated = if (existing.any { it.locator.bookmarkIdentifier() == identifier }) {
            existing.filterNot { it.locator.bookmarkIdentifier() == identifier }
        } else {
            listOf(EpubBookmark(locator.boundedText(), System.currentTimeMillis())) + existing
        }.take(MAX_BOOKMARKS)
        save(publicationKey, updated)
        return updated
    }

    private fun save(publicationKey: String, bookmarks: List<EpubBookmark>) {
        val values = JSONArray()
        bookmarks.forEach { bookmark ->
            values.put(JSONObject().apply {
                put("locator", bookmark.locator.toJSON())
                put("created_at", bookmark.createdAt)
            })
        }
        preferences.edit().putString(publicationKey, values.toString()).apply()
    }
}

private const val MAX_BOOKMARKS = 200
private const val MAX_ENCODED_SIZE = 4 * 1024 * 1024

/** Reads the saved bookmarks, skipping an entry that cannot be read so one bad bookmark never hides (or erases) the rest. */
internal fun parseBookmarks(encoded: String): List<EpubBookmark> =
    readSavedEntries(encoded) { value ->
        EpubBookmark(Locator.fromJSON(value.getJSONObject("locator"))!!, value.optLong("created_at"))
    }.sortedByDescending(EpubBookmark::createdAt)

internal fun <T> readSavedEntries(encoded: String, decode: (JSONObject) -> T): List<T> = runCatching {
    require(encoded.length <= MAX_ENCODED_SIZE)
    val values = JSONArray(encoded)
    (0 until minOf(values.length(), MAX_BOOKMARKS)).mapNotNull { index ->
        runCatching { decode(values.getJSONObject(index)) }.getOrNull()
    }
}.getOrDefault(emptyList())

fun Locator.bookmarkIdentifier(): String = buildString {
    append(href)
    append('|')
    append(locations.fragments.joinToString("/"))
    append('|')
    append(locations.progression ?: "")
}

private fun Locator.boundedText(): Locator = copy(
    text = Locator.Text(
        before = text.before?.takeLast(256),
        highlight = text.highlight?.take(512),
        after = text.after?.take(256),
    ),
)
