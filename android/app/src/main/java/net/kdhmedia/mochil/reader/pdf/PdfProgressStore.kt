package net.kdhmedia.mochil.reader.pdf

import android.content.Context
import net.kdhmedia.mochil.reader.fixedLayoutStorageKey

internal data class PdfLocalSyncState(
    val revision: Long,
    val page: Int?,
    val pendingPage: Int? = null,
)

internal class PdfProgressStore(context: Context) {
    private val preferences = context.getSharedPreferences("pdf_progress_sync", Context.MODE_PRIVATE)

    fun load(publicationKey: String): PdfLocalSyncState = PdfLocalSyncState(
        revision = preferences.getLong("revision:$publicationKey", 0L),
        page = preferences.getInt("page:$publicationKey", -1).takeIf { it >= 0 },
        pendingPage = preferences.getInt("pending_page:$publicationKey", -1).takeIf { it >= 0 },
    )

    fun migrateLegacyKey(legacyKey: String, publicationKey: String) {
        if (legacyKey == publicationKey || preferences.contains("revision:$publicationKey") || preferences.contains("page:$publicationKey") || preferences.contains("pending_page:$publicationKey")) return
        val editor = preferences.edit()
        var migrated = false
        listOf("revision", "page", "pending_page").forEach { field ->
            val old = "$field:$legacyKey"
            if (!preferences.contains(old)) return@forEach
            migrated = true
            if (field == "revision") editor.putLong("$field:$publicationKey", preferences.getLong(old, 0L))
            else editor.putInt("$field:$publicationKey", preferences.getInt(old, -1))
            editor.remove(old)
        }
        if (migrated) editor.apply()
    }

    fun markPending(publicationKey: String, page: Int) {
        preferences.edit().putInt("pending_page:$publicationKey", page.coerceAtLeast(0)).apply()
    }

    fun markSynchronized(publicationKey: String, revision: Long, page: Int, pendingPage: Int? = null) {
        val editor = preferences.edit()
            .putLong("revision:$publicationKey", revision.coerceAtLeast(0L))
            .putInt("page:$publicationKey", page.coerceAtLeast(0))
        if (pendingPage == null) editor.remove("pending_page:$publicationKey")
        else editor.putInt("pending_page:$publicationKey", pendingPage.coerceAtLeast(0))
        editor.apply()
    }
}

internal enum class PdfOpenDecision { KEEP_LOCAL, USE_REMOTE, ASK }

/** Length-prefixing prevents `a:b`/`c` and `a`/`b:c` from sharing local sync state. */
internal fun fixedLayoutSyncKey(libraryId: String, bookId: String): String =
    fixedLayoutStorageKey(libraryId, bookId)

internal fun decidePdfOpenPosition(
    localPage: Int,
    remotePage: Int,
    remoteRevision: Long,
    synchronized: PdfLocalSyncState,
): PdfOpenDecision {
    if (localPage == remotePage || remoteRevision <= synchronized.revision) return PdfOpenDecision.KEEP_LOCAL
    if (synchronized.page != null && localPage == synchronized.page) return PdfOpenDecision.USE_REMOTE
    return PdfOpenDecision.ASK
}
