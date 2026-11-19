package net.kdhmedia.mochil.webdav

import android.content.Context
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.put
import net.kdhmedia.mochil.auth.EncryptedPayloadStore

internal data class WebDavIndex(val connectionId: String, val credentialKey: String, val entries: List<WebDavEntry>, val scannedAt: Long, val complete: Boolean)

internal class WebDavIndexStore(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "webdav_media_index", "mochil_webdav_media_index")

    @Synchronized fun get(connection: WebDavConnection): WebDavIndex? = decodeWebDavIndexes(encrypted.load()).firstOrNull {
        it.connectionId == connection.id && it.credentialKey == webDavCredentialKey(connection)
    }

    @Synchronized fun put(index: WebDavIndex) {
        val values = (decodeWebDavIndexes(encrypted.load()).filterNot { it.connectionId == index.connectionId } + index)
            .sortedByDescending(WebDavIndex::scannedAt).take(MAX_CONNECTIONS)
        val payload = encodeWebDavIndexes(values)
        require(payload.size <= MAX_INDEX_BYTES) { "This WebDAV index is too large to save on this device." }
        encrypted.save(payload)
    }

    @Synchronized fun remove(connectionId: String) {
        val remaining = decodeWebDavIndexes(encrypted.load()).filterNot { it.connectionId == connectionId }
        if (remaining.isEmpty()) encrypted.clear() else encrypted.save(encodeWebDavIndexes(remaining))
    }
}

internal fun encodeWebDavIndexes(values: List<WebDavIndex>): ByteArray = buildJsonArray {
    values.take(MAX_CONNECTIONS).forEach { index -> add(buildJsonObject {
        put("connection", index.connectionId); put("credential", index.credentialKey); put("scanned", index.scannedAt); put("complete", index.complete)
        put("entries", buildJsonArray { index.entries.take(MAX_INDEX_ENTRIES).forEach { entry -> add(buildJsonObject {
            put("href", entry.href); put("name", entry.name); put("length", entry.contentLength); put("type", entry.contentType)
            put("etag", entry.etag); put("modified", entry.modified)
        }) } })
    }) }
}.toString().encodeToByteArray()

internal fun decodeWebDavIndexes(payload: ByteArray?): List<WebDavIndex> = runCatching {
    if (payload == null || payload.size > MAX_INDEX_BYTES) return emptyList()
    val root = Json.parseToJsonElement(payload.decodeToString()) as? JsonArray ?: return emptyList()
    root.take(MAX_CONNECTIONS).mapNotNull { value -> (value as? JsonObject)?.run {
        val entriesJson = this["entries"] as? JsonArray ?: return@mapNotNull null
        val entries = entriesJson.take(MAX_INDEX_ENTRIES).mapNotNull { entryValue -> (entryValue as? JsonObject)?.run {
            val href = string("href").take(4_096); val name = string("name").take(500)
            if (href.isBlank() || name.isBlank()) return@mapNotNull null
            WebDavEntry(href, name, false, long("length"), string("type").take(160), string("etag").take(500), string("modified").take(160))
        } }.filter(WebDavEntry::supported).distinctBy(WebDavEntry::href)
        WebDavIndex(string("connection").take(64), string("credential").take(64), entries, long("scanned"), boolean("complete"))
    } }
}.getOrDefault(emptyList())

private fun JsonObject.string(name: String) = (this[name] as? JsonPrimitive)?.contentOrNull.orEmpty()
private fun JsonObject.long(name: String) = string(name).toLongOrNull()?.coerceAtLeast(0) ?: 0
private fun JsonObject.boolean(name: String) = string(name).toBooleanStrictOrNull() ?: false

internal const val MAX_INDEX_ENTRIES = 10_000
internal const val MAX_INDEX_BYTES = 4 * 1024 * 1024
private const val MAX_CONNECTIONS = 20
