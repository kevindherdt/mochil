package net.kdhmedia.mochil.wiki

import org.json.JSONArray
import org.json.JSONObject

internal fun encodeWikiOfflineSnapshot(snapshot: WikiOfflineSnapshot): ByteArray? =
    encodeOfflineSnapshot(snapshot, trimOldest = false)

internal fun boundedWikiOfflinePayload(snapshot: WikiOfflineSnapshot): ByteArray? =
    encodeOfflineSnapshot(snapshot, trimOldest = true)

/** Account for encoded UTF-8 bytes incrementally instead of re-encoding shrinking vaults. */
private fun encodeOfflineSnapshot(snapshot: WikiOfflineSnapshot, trimOldest: Boolean): ByteArray? {
    if (snapshot.libraryId.isBlank() || snapshot.entries.size > MAX_OFFLINE_ENTRIES ||
        (!trimOldest && snapshot.pages.size > MAX_OFFLINE_PAGES)) return null
    val entries = JSONArray()
    snapshot.entries.forEach { entry -> entries.put(JSONObject()
        .put("kind", entry.kind).put("id", entry.id).put("path", entry.path)
        .put("title", entry.title).put("format", entry.format).put("revision", entry.revision)) }
    val pages = JSONArray()
    val value = JSONObject()
        .put("library_id", snapshot.libraryId).put("entries", entries)
        .put("selected_path", snapshot.selectedPath).put("pages", pages).put("saved_at", snapshot.savedAt)
    var bytes = value.toString().encodeToByteArray().size
    if (bytes > MAX_OFFLINE_PAYLOAD_BYTES) return null
    for (cached in snapshot.pages.take(MAX_OFFLINE_PAGES)) {
        val encoded = encodeWikiOfflinePage(cached)
        if (encoded == null) {
            if (trimOldest) break else return null
        }
        val additional = encoded.toString().encodeToByteArray().size + if (pages.length() == 0) 0 else 1
        if (additional > MAX_OFFLINE_PAYLOAD_BYTES - bytes) {
            if (trimOldest) break else return null
        }
        pages.put(encoded)
        bytes += additional
    }
    return value.toString().encodeToByteArray().takeIf { it.size <= MAX_OFFLINE_PAYLOAD_BYTES }
}

internal fun encodeWikiOfflinePage(cached: WikiOfflinePage): JSONObject? {
    if (cached.path.isBlank() || cached.page.id.isBlank() || cached.page.content.length > MAX_OFFLINE_PAGE_CHARS) return null
    return JSONObject()
        .put("path", cached.path).put("saved_at", cached.savedAt)
        .put("page", JSONObject().put("id", cached.page.id).put("title", cached.page.title)
            .put("content", cached.page.content).put("revision", cached.page.revision))
}
