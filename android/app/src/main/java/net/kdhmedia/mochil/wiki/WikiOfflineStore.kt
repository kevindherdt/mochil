package net.kdhmedia.mochil.wiki

import android.content.Context
import java.security.MessageDigest
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import org.json.JSONArray
import org.json.JSONObject

internal data class WikiOfflinePage(val path: String, val page: WikiPage, val savedAt: Long)

internal data class WikiOfflineSnapshot(
    val libraryId: String,
    val entries: List<WikiEntry>,
    val selectedPath: String,
    val pages: List<WikiOfflinePage>,
    val savedAt: Long,
) {
    fun page(id: String): WikiOfflinePage? = pages.firstOrNull { it.page.id == id }
    fun selectedPage(): WikiOfflinePage? = pages.firstOrNull { it.path == selectedPath }
}

internal fun mergeWikiOfflineSnapshots(previous: WikiOfflineSnapshot?, incoming: WikiOfflineSnapshot): WikiOfflineSnapshot {
    val compatible = previous?.takeIf { it.libraryId == incoming.libraryId }
    val latest = if (compatible == null || incoming.savedAt >= compatible.savedAt) incoming else compatible
    val pages = (incoming.pages + compatible?.pages.orEmpty()).sortedByDescending(WikiOfflinePage::savedAt)
        .distinctBy { it.page.id }
        // Resolve newest content first, then protect the page the reader selected.
        .sortedByDescending { it.path == latest.selectedPath }.take(MAX_OFFLINE_PAGES)
    return latest.copy(pages = pages)
}


internal fun decodeWikiOfflineSnapshot(bytes: ByteArray): WikiOfflineSnapshot? = runCatching {
    if (bytes.size > MAX_OFFLINE_PAYLOAD_BYTES) return null
    val value = JSONObject(bytes.decodeToString())
    val entriesJSON = value.getJSONArray("entries")
    if (entriesJSON.length() > MAX_OFFLINE_ENTRIES) return null
    val entries = List(entriesJSON.length()) { index -> entriesJSON.getJSONObject(index).let { entry ->
        WikiEntry(entry.getString("kind"), entry.optString("id"), entry.getString("path"), entry.optString("title"), entry.optString("format"), revision = entry.optString("revision"))
    } }
    val pagesJSON = value.optJSONArray("pages") ?: JSONArray().also { legacy ->
        value.optJSONObject("page")?.let { page ->
            legacy.put(JSONObject().put("path", value.optString("selected_path")).put("saved_at", value.optLong("saved_at")).put("page", page))
        }
    }
    if (pagesJSON.length() > MAX_OFFLINE_PAGES) return null
    val pages = List(pagesJSON.length()) { index -> pagesJSON.getJSONObject(index).let { cached ->
        val page = cached.getJSONObject("page")
        WikiOfflinePage(
            cached.getString("path"),
            WikiPage(page.getString("id"), page.optString("title", "Untitled"), page.getString("content"), page.optString("revision"), false),
            cached.optLong("saved_at"),
        )
    } }
    if (pages.any { it.path.isBlank() || it.page.id.isBlank() || it.page.content.length > MAX_OFFLINE_PAGE_CHARS }) return null
    WikiOfflineSnapshot(value.getString("library_id"), entries, value.optString("selected_path"), pages, value.optLong("saved_at"))
}.getOrNull()

/** A small encrypted recent-page set gives offline reading without growing into a second vault. */
internal class WikiOfflineStore(context: Context) {
    private val applicationContext = context.applicationContext
    private val legacy = EncryptedPayloadStore(applicationContext, LEGACY_PREFERENCES, LEGACY_KEY_ALIAS)
    private val index = applicationContext.getSharedPreferences(INDEX_PREFERENCES, Context.MODE_PRIVATE)

    fun save(ownerId: String, snapshot: WikiOfflineSnapshot) = synchronized(storageLock) {
        if (ownerId.isBlank()) return@synchronized
        clearUnowned(snapshot.libraryId)
        val identity = wikiOfflineStorageIdentity(ownerId, snapshot.libraryId)
        val encrypted = encrypted(identity)
        val previous = encrypted.load()?.let(::decodeWikiOfflineSnapshot)
        val payload = boundedWikiOfflinePayload(mergeWikiOfflineSnapshots(previous, snapshot))
            ?: throw java.io.IOException("This vault exceeds the offline cache limits")
        PerformanceDiagnostics.measure("storage", "save_wiki_cache") {
            val current = index.getString(INDEX_KEY, "").orEmpty().split(',').filter(String::isNotBlank)
            val order = updateWikiOfflineStorageOrder(current, identity)
            commitWikiOfflineCache(
                writePayload = { encrypted.save(payload) },
                writeIndex = { index.edit().putString(INDEX_KEY, order.kept.joinToString(",")).commit() },
                evict = { order.evicted.forEach { encrypted(it).clear() } },
            )
        }
    }

    /** Automatic read caching must not make opening an online page fail. */
    fun saveBestEffort(ownerId: String, snapshot: WikiOfflineSnapshot) {
        try { save(ownerId, snapshot) }
        catch (_: Exception) { android.util.Log.w("WikiOfflineStore", "Could not update the offline Wiki cache") }
    }
    fun load(ownerId: String, libraryId: String): WikiOfflineSnapshot? = synchronized(storageLock) { PerformanceDiagnostics.measure("storage", "load_wiki_cache") {
        clearUnowned(libraryId)
        if (ownerId.isBlank()) null else encrypted(wikiOfflineStorageIdentity(ownerId, libraryId)).load()?.let(::decodeWikiOfflineSnapshot)?.takeIf { it.libraryId == libraryId }
    } }

    private fun clearUnowned(libraryId: String) {
        legacy.clear()
        encrypted(wikiLegacyStorageIdentity(libraryId)).clear()
    }

    private fun encrypted(identity: String) =
        EncryptedPayloadStore(applicationContext, "wiki_offline_snapshot_$identity", "mochil_wiki_offline_snapshot_$identity", MAX_OFFLINE_PAYLOAD_BYTES)

    private companion object {
        val storageLock = Any()
        const val LEGACY_PREFERENCES = "wiki_offline_snapshot"
        const val LEGACY_KEY_ALIAS = "mochil_wiki_offline_snapshot"
        const val INDEX_PREFERENCES = "wiki_offline_snapshot_index"
        const val INDEX_KEY = "recent_vaults"
    }
}

/** Keeps vault names and server-facing identifiers out of preferences and keystore aliases. */
internal fun wikiOfflineStorageIdentity(ownerId: String, libraryId: String): String = MessageDigest.getInstance("SHA-256")
    .digest("${ownerId.trim()}\u0000${libraryId.trim()}".encodeToByteArray())
    .take(12)
    .joinToString("") { byte -> "%02x".format(byte.toInt() and 0xff) }

internal fun wikiLegacyStorageIdentity(libraryId: String): String = MessageDigest.getInstance("SHA-256")
    .digest(libraryId.trim().encodeToByteArray()).take(12)
    .joinToString("") { byte -> "%02x".format(byte.toInt() and 0xff) }

internal data class WikiOfflineStorageOrder(val kept: List<String>, val evicted: List<String>)

internal fun updateWikiOfflineStorageOrder(current: List<String>, used: String): WikiOfflineStorageOrder {
    val ordered = (listOf(used) + current).filter { it.matches(WIKI_STORAGE_IDENTITY) }.distinct()
    return WikiOfflineStorageOrder(ordered.take(MAX_OFFLINE_VAULTS), ordered.drop(MAX_OFFLINE_VAULTS))
}

internal const val MAX_OFFLINE_ENTRIES = 5_000
internal const val MAX_OFFLINE_PAGES = 128
internal const val MAX_OFFLINE_PAGE_CHARS = 2_000_000
internal const val MAX_OFFLINE_PAYLOAD_BYTES = 3_000_000
private const val MAX_OFFLINE_VAULTS = 4
private val WIKI_STORAGE_IDENTITY = Regex("[0-9a-f]{24}")
