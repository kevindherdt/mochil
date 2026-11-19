@file:android.annotation.SuppressLint("ApplySharedPref") // A completed publication is not exposed until its durable index is committed.
package net.kdhmedia.mochil.catalogue

import android.content.Context
import android.content.SharedPreferences
import java.io.File
import java.util.concurrent.ConcurrentHashMap
import org.json.JSONArray
import org.json.JSONObject

data class DownloadedPublication(
    val libraryId: String,
    val bookId: String,
    val title: String,
    val authors: String,
    val format: String,
    val mediaType: String,
    val path: String,
    val size: Long,
    val series: String = "",
    val seriesIndex: Double? = null,
    val language: String = "",
    val description: String = "",
    val externalUri: String? = null,
)

/**
 * Thrown by [PublicationDownloadStore.record]/[PublicationDownloadStore.remove] when the on-disk
 * index cannot be parsed. The raw preference value is left exactly as it was found -- the caller
 * must not treat this as "the index is empty", and nothing merges a new/removed entry onto a
 * synthesised empty list, because that would silently and permanently destroy whatever was in the
 * corrupt file. This does not make the underlying file/mirror writes transactional: a crash between
 * the file move and this commit can still leave the file and the index disagreeing, same as before.
 */
class PublicationIndexCorruptException(message: String, cause: Throwable? = null) : IllegalStateException(message, cause)

class PublicationDownloadStore(
    private val context: Context,
    private val preferences: SharedPreferences = context.getSharedPreferences(PREFERENCE_NAME, Context.MODE_PRIVATE),
) {
    // @Synchronized locks on `this`, but every call site (OpdsViewModel, PublicationOpener,
    // LibraryViewModel) constructs a fresh PublicationDownloadStore(context) per use. Two such
    // instances share the on-disk index but never shared a monitor, so a decode-modify-commit cycle
    // on one instance could run entirely between another instance's decode and commit and be
    // overwritten without a trace -- a lost update, not a crash, so it never showed up in logs.
    // Keying the lock by the owning application Context's identity plus the preference file name
    // makes every instance backed by the same index serialize through the same monitor, while still
    // letting instances backed by a genuinely different context (e.g. isolated test contexts) run
    // independently instead of contending on an unrelated lock.
    private val lock: Any = locksByIndex.getOrPut(IndexKey(System.identityHashCode(context.applicationContext), PREFERENCE_NAME)) { Any() }

    fun list(): List<DownloadedPublication> = synchronized(lock) {
        decodeOrEmpty().filter { File(it.path).isFile }.sortedBy { it.title.lowercase() }
    }

    fun find(libraryId: String, bookId: String): DownloadedPublication? = synchronized(lock) {
        decodeOrEmpty().firstOrNull { it.libraryId == libraryId && it.bookId == bookId }
    }

    /**
     * Records [book] as downloaded to [file]. Returns false, without throwing, when the commit to
     * disk failed -- the file is on disk either way, but a caller that checks the result can now
     * tell a silent persistence failure from a real success, where before the `commit()` boolean
     * was discarded and every caller believed the index write always succeeded.
     *
     * @throws PublicationIndexCorruptException if the existing index cannot be parsed; see the
     * class doc for why this does not fall back to an empty index.
     */
    fun record(book: BookSummary, file: File, externalUri: String? = null): Boolean = synchronized(lock) {
        val decoded = decodeOrThrow()
        val previous = decoded.firstOrNull { it.libraryId == book.libraryId && it.bookId == book.id }
        val items = decoded.filterNot { it.libraryId == book.libraryId && it.bookId == book.id } + DownloadedPublication(
            libraryId = book.libraryId,
            bookId = book.id,
            title = book.title,
            authors = book.authors.joinToString(),
            format = book.format,
            mediaType = book.mediaType,
            path = file.absolutePath,
            size = file.length(),
            series = book.series,
            seriesIndex = book.seriesIndex,
            language = book.language,
            description = book.description,
            externalUri = externalUri ?: previous?.externalUri,
        )
        save(items)
    }

    /**
     * Deletes [item]'s file and, only if the index can be safely rewritten, drops it from the
     * index too. Returns true only when the file is gone AND the index commit succeeded; a corrupt
     * index or a failed commit now returns false even though the file may already be gone, so a
     * caller cannot mistake a half-applied removal for a complete one. This still does not make the
     * file deletion and the index update one atomic transaction -- a crash between them can still
     * leave a dangling index entry whose file no longer exists (already handled by [list] filtering
     * missing files), the same gap that existed before this change.
     */
    fun remove(item: DownloadedPublication): Boolean = synchronized(lock) {
        val decoded = decode()
        // The caller's [item] is a snapshot and may be stale (the index can have been repointed since it was
        // read). Everything below acts on the CURRENT indexed entry for (libraryId, bookId), re-read here
        // under the lock; only item.libraryId/bookId are trusted from the snapshot.
        val all = (decoded as? DecodeResult.Loaded)?.items.orEmpty()
        val current = all.firstOrNull { it.libraryId == item.libraryId && it.bookId == item.bookId }
        val target = when {
            current != null -> current
            // Unreadable index: we cannot know the current entry, so keep the original behaviour (delete the
            // caller's path, report failure, leave the raw index untouched).
            decoded is DecodeResult.Corrupt -> item
            // Key no longer indexed (removed or never there). Safest is to delete NOTHING: the stale path may
            // now be a file a concurrent download has moved into place but not yet recorded, and an entry that
            // was already removed has already had its file handled. Nothing to commit either; report success.
            else -> return@synchronized true
        }
        // Legacy builds could point several entries at one file (and one mirror). Only delete what no
        // remaining entry still references.
        val others = all.filterNot { it.libraryId == item.libraryId && it.bookId == item.bookId }
        val file = File(target.path)
        val fileShared = others.any { File(it.path).absoluteFile == file.absoluteFile }
        val fileRemoved = fileShared || !file.exists() || file.delete()
        if (!fileRemoved) return@synchronized false
        // The mirror belongs to the removed entry: it goes unless a remaining entry holds the same mirror URI,
        // even when the private file itself is kept for a sibling that shares it.
        if (target.externalUri != null && others.none { it.externalUri == target.externalUri }) {
            DownloadLocationStore(context).delete(target.externalUri)
        }
        when (decoded) {
            is DecodeResult.Corrupt -> false
            is DecodeResult.Loaded -> save(others)
        }
    }

    private fun decodeOrEmpty(): List<DownloadedPublication> = when (val result = decode()) {
        is DecodeResult.Loaded -> result.items
        is DecodeResult.Corrupt -> emptyList()
    }

    private fun decodeOrThrow(): List<DownloadedPublication> = when (val result = decode()) {
        is DecodeResult.Loaded -> result.items
        is DecodeResult.Corrupt -> throw PublicationIndexCorruptException(
            "The downloaded-publication index (preference '$PREFERENCE_NAME.$KEY_ITEMS') is corrupt; refusing to overwrite it.",
            result.cause,
        )
    }

    private sealed class DecodeResult {
        data class Loaded(val items: List<DownloadedPublication>) : DecodeResult()
        data class Corrupt(val cause: Throwable) : DecodeResult()
    }

    private fun decode(): DecodeResult = runCatching {
        val values = JSONArray(preferences.getString(KEY_ITEMS, "[]"))
        DecodeResult.Loaded(
            List(values.length()) { index ->
                values.getJSONObject(index).run {
                    DownloadedPublication(
                        libraryId = getString("library_id"), bookId = getString("book_id"), title = getString("title"),
                        authors = optString("authors"), format = optString("format"), mediaType = optString("media_type"),
                        path = getString("path"), size = optLong("size"),
                        series = optString("series"),
                        seriesIndex = if (has("series_index") && !isNull("series_index")) getDouble("series_index") else null,
                        language = optString("language"), description = optString("description"),
                        externalUri = optString("external_uri").takeIf(String::isNotBlank),
                    )
                }
            },
        )
    }.getOrElse { DecodeResult.Corrupt(it) }

    /** Returns whether the index commit to disk succeeded; callers must not assume it did. */
    private fun save(items: List<DownloadedPublication>): Boolean {
        val values = JSONArray()
        items.forEach { item ->
            values.put(JSONObject().apply {
                put("library_id", item.libraryId); put("book_id", item.bookId); put("title", item.title)
                put("authors", item.authors); put("format", item.format); put("media_type", item.mediaType)
                put("path", item.path); put("size", item.size)
                put("series", item.series); put("series_index", item.seriesIndex)
                put("language", item.language); put("description", item.description)
                put("external_uri", item.externalUri)
            })
        }
        return preferences.edit().putString(KEY_ITEMS, values.toString()).commit()
    }

    private companion object {
        const val PREFERENCE_NAME = "publication_downloads"
        const val KEY_ITEMS = "items"
        data class IndexKey(val contextIdentity: Int, val preferenceName: String)
        val locksByIndex = ConcurrentHashMap<IndexKey, Any>()
    }
}

fun DownloadedPublication.asBookSummary() = BookSummary(
    id = bookId, libraryId = libraryId, title = title,
    authors = authors.split(',').map(String::trim).filter(String::isNotBlank),
    series = series, seriesIndex = seriesIndex, format = format, mediaType = mediaType,
    language = language, description = description, size = size,
)
