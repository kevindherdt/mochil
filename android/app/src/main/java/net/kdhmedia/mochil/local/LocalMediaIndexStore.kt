package net.kdhmedia.mochil.local

import android.content.Context
import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.put

internal class LocalMediaIndexStore(context: Context) {
    private val file = File(context.noBackupFilesDir, "local-media-index.json")

    @Synchronized fun all(sourceUris: Set<String>): List<LocalMediaItem> = decodeLocalMediaIndex(read()).filter { it.sourceUri in sourceUris }

    @Synchronized fun replace(sourceUri: String, items: List<LocalMediaItem>) {
        val merged = (decodeLocalMediaIndex(read()).filterNot { it.sourceUri == sourceUri } + items).take(MAX_INDEX_ITEMS)
        write(encodeLocalMediaIndex(merged))
    }

    @Synchronized fun remove(sourceUri: String) = write(encodeLocalMediaIndex(decodeLocalMediaIndex(read()).filterNot { it.sourceUri == sourceUri }))

    private fun read(): ByteArray? = file.takeIf { it.isFile && it.length() in 1..MAX_INDEX_BYTES.toLong() }?.readBytes()

    private fun write(payload: ByteArray) {
        if (payload.size > MAX_INDEX_BYTES) return
        file.parentFile?.mkdirs()
        val temporary = File(file.parentFile, ".${file.name}.part")
        try {
            temporary.writeBytes(payload)
            if (!temporary.renameTo(file)) temporary.copyTo(file, overwrite = true)
        } finally { temporary.delete() }
    }
}

internal fun encodeLocalMediaIndex(items: List<LocalMediaItem>): ByteArray = buildJsonArray { items.take(MAX_INDEX_ITEMS).forEach { item -> add(buildJsonObject {
    put("source", item.sourceUri); put("document", item.documentUri); put("name", item.name); put("extension", item.extension); put("mime", item.mimeType)
    put("size", item.size); put("modified", item.lastModified); put("title", item.embeddedTitle); put("artist", item.artist); put("album", item.album); put("duration", item.durationSeconds)
}) } }.toString().encodeToByteArray()

internal fun decodeLocalMediaIndex(bytes: ByteArray?): List<LocalMediaItem> = runCatching {
    if (bytes == null || bytes.size > MAX_INDEX_BYTES) return emptyList()
    val root = Json.parseToJsonElement(bytes.decodeToString()) as? JsonArray ?: return emptyList()
    root.take(MAX_INDEX_ITEMS).mapNotNull { value ->
        val item = value as? JsonObject ?: return@mapNotNull null
        val source = item.string("source").take(MAX_URI); val document = item.string("document").take(MAX_URI)
        val name = item.string("name").take(MAX_NAME); val extension = item.string("extension").take(12).lowercase()
        val kind = localMediaKind(extension) ?: return@mapNotNull null
        if (source.isBlank() || document.isBlank() || name.isBlank()) return@mapNotNull null
        LocalMediaItem(source, document, name, extension, item.string("mime").take(160), item.long("size"), kind,
            item.long("modified"), item.string("title").take(MAX_NAME), item.string("artist").take(MAX_NAME), item.string("album").take(MAX_NAME), item.double("duration"))
    }
}.getOrDefault(emptyList())

private fun JsonObject.string(name: String) = (this[name] as? JsonPrimitive)?.contentOrNull.orEmpty()
private fun JsonObject.long(name: String) = string(name).toLongOrNull()?.coerceAtLeast(0) ?: 0
private fun JsonObject.double(name: String) = string(name).toDoubleOrNull()?.takeIf { it.isFinite() && it >= 0 } ?: 0.0
internal const val MAX_INDEX_ITEMS = 50_000
internal const val MAX_INDEX_BYTES = 16 * 1024 * 1024
private const val MAX_URI = 8_192
private const val MAX_NAME = 1_000
