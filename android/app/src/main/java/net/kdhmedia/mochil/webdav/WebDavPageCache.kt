package net.kdhmedia.mochil.webdav

import android.content.Context
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import org.json.JSONArray
import org.json.JSONObject

internal data class CachedWebDavPage(val connectionId: String, val credentialKey: String, val page: WebDavPage, val usedAt: Long)

internal class WebDavPageCache(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "webdav_page_cache", "mochil_webdav_page_cache")
    // The on-disk blob is bounded (~2MB) but a page navigation can happen many times a minute; a full
    // decrypt + JSON parse (and, on write, re-serialize) on every one of them is wasted work once this
    // instance (held at WebDavViewModel scope, i.e. for the life of the WebDAV screen session) has already
    // seen the blob once. Keep the decoded form in memory and only touch the encrypted store again on a write.
    private val memory = DecodedWebDavCacheMemory(load = encrypted::load, save = encrypted::save)

    @Synchronized fun get(connection: WebDavConnection, url: String): WebDavPage? = memory.decoded().firstOrNull {
        it.connectionId == connection.id && it.credentialKey == webDavCredentialKey(connection) && it.page.url == url
    }?.page

    @Synchronized fun put(connection: WebDavConnection, page: WebDavPage) {
        val identity = webDavCredentialKey(connection)
        val current = memory.decoded().filterNot { it.connectionId == connection.id && it.page.url == page.url }
        val bounded = (current + CachedWebDavPage(connection.id, identity, page.copy(entries = page.entries.take(MAX_ENTRIES_PER_PAGE), notModified = false), System.currentTimeMillis()))
            .sortedByDescending(CachedWebDavPage::usedAt).take(MAX_PAGES)
        val payload = encodeWebDavCache(bounded)
        if (payload.size <= MAX_PAYLOAD_BYTES) memory.replace(bounded, payload)
    }

    @Synchronized fun removeConnection(connectionId: String) {
        val bounded = memory.decoded().filterNot { it.connectionId == connectionId }
        memory.replace(bounded, encodeWebDavCache(bounded))
    }

    private companion object { const val MAX_PAGES = 24; const val MAX_ENTRIES_PER_PAGE = 500; const val MAX_PAYLOAD_BYTES = 2 * 1024 * 1024 }
}

/**
 * The decode/encode round trip itself (free functions below), kept separate from any Android
 * dependency so the memoization can be exercised in a plain JVM unit test: [decoded] only calls
 * [load] (and therefore only decrypts/parses) the first time, or again after an explicit [replace]
 * miss; every other call reuses the in-memory list.
 */
internal class DecodedWebDavCacheMemory(private val load: () -> ByteArray?, private val save: (ByteArray) -> Boolean) {
    private var cached: List<CachedWebDavPage>? = null

    fun decoded(): List<CachedWebDavPage> = cached ?: decodeWebDavCache(load()).also { cached = it }

    /** Persists [payload] and, only if that succeeds, adopts [list] as the in-memory cache so the
     *  next [decoded] does not need to re-read and re-parse what was just written. */
    fun replace(list: List<CachedWebDavPage>, payload: ByteArray) {
        if (save(payload)) cached = list
    }
}

internal fun encodeWebDavCache(values: List<CachedWebDavPage>): ByteArray = JSONArray().apply { values.forEach { cached -> put(JSONObject().apply {
    put("connection", cached.connectionId); put("credential", cached.credentialKey); put("url", cached.page.url); put("validator", cached.page.validator); put("used", cached.usedAt)
    put("entries", JSONArray().apply { cached.page.entries.forEach { entry -> put(JSONObject().apply {
        put("href", entry.href); put("name", entry.name); put("collection", entry.collection); put("length", entry.contentLength)
        put("type", entry.contentType); put("etag", entry.etag); put("modified", entry.modified)
    }) } })
}) } }.toString().encodeToByteArray()

internal fun decodeWebDavCache(payload: ByteArray?): List<CachedWebDavPage> = runCatching {
    if (payload == null || payload.size > 2 * 1024 * 1024) return emptyList()
    val root = JSONArray(payload.decodeToString())
    List(minOf(root.length(), 24)) { index -> root.getJSONObject(index).run {
        val entriesJson = getJSONArray("entries")
        val entries = List(minOf(entriesJson.length(), 500)) { entryIndex -> entriesJson.getJSONObject(entryIndex).run {
            WebDavEntry(getString("href").take(4096), getString("name").take(500), optBoolean("collection"), optLong("length").coerceAtLeast(0), optString("type").take(160), optString("etag").take(500), optString("modified").take(160))
        } }.filter(WebDavEntry::supported)
        CachedWebDavPage(getString("connection").take(64), optString("credential").take(64), WebDavPage(getString("url").take(4096), entries, optString("validator").take(500)), optLong("used"))
    } }
}.getOrDefault(emptyList())
