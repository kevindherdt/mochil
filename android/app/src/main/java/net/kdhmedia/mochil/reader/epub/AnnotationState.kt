package net.kdhmedia.mochil.reader.epub

import android.content.Context
import android.util.AtomicFile
import java.io.File
import java.security.MessageDigest
import java.util.UUID
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.network.readBoundedResponseText
import org.json.JSONArray
import org.json.JSONObject
import org.readium.r2.shared.publication.Locator

data class LocalAnnotationState(
    val serverRevision: Long = 0,
    val items: Map<String, SyncedAnnotation> = emptyMap(),
    val pendingBaseRevisions: Map<String, Long> = emptyMap(),
    val conflicts: Map<String, AnnotationConflict> = emptyMap(),
) {
    val visible: List<SyncedAnnotation>
        get() = items.values
            .filterNot(SyncedAnnotation::deleted)
            .sortedByDescending(SyncedAnnotation::updatedAt)
            .distinctBy { it.semanticIdentity() ?: "id:${it.id}" }

    fun createHighlight(locator: Locator, deviceId: String, spineIndex: Int, now: Long = System.currentTimeMillis()): LocalAnnotationState {
        val quote = locator.text.highlight.orEmpty().trim().boundedUtf8(MAX_QUOTE_BYTES)
        if (quote.isBlank() || items.size >= MAX_ITEMS) return this
        val bounded = locator.boundedAnnotationLocator()
        val anchor = ServerTextAnchor(
            prefix = bounded.text.before.orEmpty().takeLast(512), exact = quote,
            suffix = bounded.text.after.orEmpty().take(512), spineIndex = spineIndex.coerceAtLeast(0),
            progress = (bounded.locations.progression ?: 0.0).coerceIn(0.0, 1.0),
        )
        if (items.values.any { !it.deleted && it.semanticIdentity() == anchor.semanticIdentity("highlight") }) return this
        val annotation = SyncedAnnotation(
            id = UUID.randomUUID().toString(), kind = "highlight", quote = quote,
            createdAt = now, updatedAt = now, deviceId = deviceId,
            locators = mapOf("readium" to bounded.toJSON().toString()),
            anchor = anchor,
        )
        return copy(items = items + (annotation.id to annotation), pendingBaseRevisions = pendingBaseRevisions + (annotation.id to 0L))
    }

    fun delete(id: String, deviceId: String, now: Long = System.currentTimeMillis()): LocalAnnotationState {
        val current = items[id] ?: return this
        if (id in conflicts) return this
        val base = pendingBaseRevisions[id] ?: current.revision
        val tombstone = current.copy(quote = "", note = "", deleted = true, deviceId = deviceId, updatedAt = now)
        return copy(items = items + (id to tombstone), pendingBaseRevisions = pendingBaseRevisions + (id to base), conflicts = conflicts - id)
    }

    fun editNote(id: String, note: String, deviceId: String, now: Long = System.currentTimeMillis()): LocalAnnotationState {
        val current = items[id] ?: return this
        if (current.deleted || id in conflicts) return this
        val boundedNote = note.trim().boundedUtf8(MAX_NOTE_BYTES)
        if (boundedNote == current.note) return this
        val base = pendingBaseRevisions[id] ?: current.revision
        val updated = current.copy(note = boundedNote, deviceId = deviceId, updatedAt = now)
        return copy(items = items + (id to updated), pendingBaseRevisions = pendingBaseRevisions + (id to base))
    }

    fun pendingChanges(): List<AnnotationChange> = pendingBaseRevisions.entries.take(MAX_BATCH_SIZE).mapNotNull { (id, revision) ->
        items[id]?.let { AnnotationChange(revision, it) }
    }

    fun withDerivedReadiumLocator(id: String, encodedLocator: String): LocalAnnotationState {
        val current = items[id] ?: return this
        if (current.deleted || "readium" in current.locators) return this
        if (encodedLocator.isBlank()) return this
        return copy(items = items + (id to current.copy(locators = current.locators + ("readium" to encodedLocator))))
    }

    fun withResolvedReadiumLocator(id: String, encodedLocator: String): LocalAnnotationState {
        val current = items[id] ?: return this
        if (current.deleted || encodedLocator.isBlank()) return this
        return copy(items = items + (id to current.copy(locators = current.locators + ("readium" to encodedLocator))))
    }

    fun mergePull(result: AnnotationSyncResult): LocalAnnotationState {
        val merged = items.toMutableMap()
        val foundConflicts = conflicts.toMutableMap()
        result.applied.forEach { remote ->
            val local = merged[remote.id]
            if (remote.id in pendingBaseRevisions && local != null && !local.sameContent(remote)) {
                foundConflicts[remote.id] = AnnotationConflict(local, remote)
            } else {
                merged[remote.id] = remote
            }
        }
        return copy(serverRevision = maxOf(serverRevision, result.revision), items = merged, conflicts = foundConflicts)
    }

    fun mergeApply(result: AnnotationSyncResult): LocalAnnotationState {
        val merged = items.toMutableMap()
        val pending = pendingBaseRevisions.toMutableMap()
        val foundConflicts = conflicts.toMutableMap()
        result.applied.forEach { saved ->
            merged[saved.id] = saved
            pending.remove(saved.id)
            foundConflicts.remove(saved.id)
        }
        result.conflicts.forEach { conflict -> foundConflicts[conflict.current.id.ifBlank { conflict.submitted.id }] = conflict }
        return copy(serverRevision = maxOf(serverRevision, result.revision), items = merged, pendingBaseRevisions = pending, conflicts = foundConflicts)
    }

    fun keepLocal(id: String): LocalAnnotationState {
        val conflict = conflicts[id] ?: return this
        val local = items[id] ?: conflict.submitted
        return copy(
            items = items + (id to local),
            pendingBaseRevisions = pendingBaseRevisions + (id to conflict.current.revision),
            conflicts = conflicts - id,
        )
    }

    fun useRemote(id: String): LocalAnnotationState {
        val conflict = conflicts[id] ?: return this
        return copy(items = items + (id to conflict.current), pendingBaseRevisions = pendingBaseRevisions - id, conflicts = conflicts - id)
    }

    companion object {
        const val MAX_ITEMS = 10_000
        const val MAX_BATCH_SIZE = 500
        const val MAX_QUOTE_BYTES = 8192
        const val MAX_NOTE_BYTES = 16384
    }
}

private fun SyncedAnnotation.semanticIdentity(): String? = anchor?.semanticIdentity(kind)

private fun ServerTextAnchor.semanticIdentity(kind: String): String? {
    val selected = stableHighlightQuery(exact)
    if (selected.isBlank() || spineIndex < 0) return null
    val before = stableHighlightQuery(prefix).takeLast(160)
    val after = stableHighlightQuery(suffix).take(160)
    return listOf(kind, spineIndex.toString(), selected, before, after).joinToString("\u0000")
}

internal interface AnnotationLocalStore {
    suspend fun load(publicationKey: String): LocalAnnotationState
    suspend fun save(publicationKey: String, state: LocalAnnotationState)
    fun needsStableLocatorRepair(publicationKey: String): Boolean
    fun markStableLocatorRepair(publicationKey: String)
}

class AnnotationStateStore(context: Context) : AnnotationLocalStore {
    private val directory = File(context.filesDir, "reader-annotations").apply { mkdirs() }
    private val migrations = context.getSharedPreferences("reader-annotation-migrations", Context.MODE_PRIVATE)

    override suspend fun load(publicationKey: String): LocalAnnotationState = withContext(Dispatchers.IO) { runCatching {
        val file = file(publicationKey)
        val encoded = file.openRead().use { stream ->
            require(file.baseFile.length() <= MAX_STATE_BYTES)
            stream.readBoundedResponseText(MAX_STATE_BYTES.toInt())
        }
        require(encoded.toByteArray().size <= MAX_STATE_BYTES)
        JSONObject(encoded).toLocalState()
    }.getOrDefault(LocalAnnotationState()) }

    override suspend fun save(publicationKey: String, state: LocalAnnotationState) = withContext(Dispatchers.IO) {
        val encoded = state.toJSON().toString()
        val bytes = encoded.encodeToByteArray()
        require(bytes.size <= MAX_STATE_BYTES)
        val file = file(publicationKey)
        val stream = file.startWrite()
        try {
            stream.write(bytes)
            stream.fd.sync()
            file.finishWrite(stream)
        } catch (failure: Throwable) {
            file.failWrite(stream)
            throw failure
        }
    }

    override fun needsStableLocatorRepair(publicationKey: String): Boolean = !migrations.getBoolean("stable-locator-v2-${digest(publicationKey)}", false)

    override fun markStableLocatorRepair(publicationKey: String) {
        migrations.edit().putBoolean("stable-locator-v2-${digest(publicationKey)}", true).apply()
    }

    private fun file(publicationKey: String): AtomicFile {
        return AtomicFile(File(directory, "${digest(publicationKey)}.json"))
    }

    private fun digest(value: String): String = MessageDigest.getInstance("SHA-256").digest(value.encodeToByteArray()).joinToString("") { "%02x".format(it) }

    private companion object { const val MAX_STATE_BYTES = 8 * 1024 * 1024L }
}

private fun LocalAnnotationState.toJSON() = JSONObject().apply {
    put("server_revision", serverRevision)
    put("items", JSONArray().apply { items.values.take(LocalAnnotationState.MAX_ITEMS).forEach { put(it.localJSON()) } })
    put("pending", JSONObject().apply { pendingBaseRevisions.forEach { (id, revision) -> put(id, revision) } })
    put("conflicts", JSONArray().apply {
        conflicts.values.take(LocalAnnotationState.MAX_BATCH_SIZE).forEach { conflict ->
            put(JSONObject().apply { put("submitted", conflict.submitted.localJSON()); put("current", conflict.current.localJSON()) })
        }
    })
}

private fun JSONObject.toLocalState(): LocalAnnotationState {
    val itemValues = optJSONArray("items") ?: JSONArray()
    val items = buildMap {
        repeat(minOf(itemValues.length(), LocalAnnotationState.MAX_ITEMS)) {
            itemValues.optJSONObject(it)?.localAnnotation()?.takeIf { value -> value.id.isNotBlank() }?.let { value -> put(value.id, value) }
        }
    }
    val pendingValues = optJSONObject("pending")
    val pending = buildMap {
        pendingValues?.keys()?.asSequence()?.take(LocalAnnotationState.MAX_BATCH_SIZE)?.forEach { id -> if (id in items) put(id, pendingValues.optLong(id)) }
    }
    val conflictValues = optJSONArray("conflicts") ?: JSONArray()
    val conflicts = buildMap {
        repeat(minOf(conflictValues.length(), LocalAnnotationState.MAX_BATCH_SIZE)) {
            val value = conflictValues.optJSONObject(it) ?: return@repeat
            val submitted = value.optJSONObject("submitted")?.localAnnotation() ?: return@repeat
            val current = value.optJSONObject("current")?.localAnnotation() ?: return@repeat
            if (submitted.id.isNotBlank()) put(submitted.id, AnnotationConflict(submitted, current))
        }
    }
    return LocalAnnotationState(optLong("server_revision"), items, pending, conflicts)
}

private fun SyncedAnnotation.localJSON() = JSONObject().apply {
    put("id", id); put("kind", kind); put("quote", quote); put("note", note); put("color", color); put("chapter", chapter)
    put("created_at", createdAt); put("updated_at", updatedAt); put("revision", revision); put("deleted", deleted); put("device_id", deviceId)
    put("locators", JSONObject(locators))
    anchor?.let { put("anchor", JSONObject().apply { put("prefix", it.prefix); put("exact", it.exact); put("suffix", it.suffix); put("spine_index", it.spineIndex); put("progress", it.progress) }) }
}

private fun JSONObject.localAnnotation(): SyncedAnnotation {
    val locatorsJSON = optJSONObject("locators")
    val locators = buildMap { locatorsJSON?.keys()?.forEach { key -> locatorsJSON.optString(key).takeIf(String::isNotBlank)?.let { put(key, it) } } }
    val anchor = optJSONObject("anchor")?.let { ServerTextAnchor(it.optString("prefix"), it.optString("exact"), it.optString("suffix"), it.optInt("spine_index"), it.optDouble("progress")) }
    return SyncedAnnotation(optString("id"), optString("kind"), optString("quote"), optString("note"), optString("color"), optString("chapter"), optLong("created_at"), optLong("updated_at"), optLong("revision"), optBoolean("deleted"), optString("device_id"), locators, anchor)
}

private fun SyncedAnnotation.sameContent(other: SyncedAnnotation): Boolean =
    copy(updatedAt = 0, revision = 0) == other.copy(updatedAt = 0, revision = 0)

private fun Locator.boundedAnnotationLocator(): Locator = copy(
    text = Locator.Text(text.before?.boundedUtf8FromEnd(512), text.highlight?.boundedUtf8(LocalAnnotationState.MAX_QUOTE_BYTES), text.after?.boundedUtf8(512)),
)

private fun String.boundedUtf8(maximumBytes: Int): String {
    if (encodeToByteArray().size <= maximumBytes) return this
    val result = StringBuilder()
    var index = 0
    var bytes = 0
    while (index < length) {
        val codePoint = codePointAt(index)
        val characters = Character.toChars(codePoint)
        val encoded = String(characters).encodeToByteArray().size
        if (bytes + encoded > maximumBytes) break
        result.append(characters)
        bytes += encoded
        index += characters.size
    }
    return result.toString()
}

private fun String.boundedUtf8FromEnd(maximumBytes: Int): String {
    if (encodeToByteArray().size <= maximumBytes) return this
    val result = StringBuilder()
    var index = length
    var bytes = 0
    while (index > 0) {
        val codePoint = codePointBefore(index)
        val characters = Character.toChars(codePoint)
        val encoded = String(characters).encodeToByteArray().size
        if (bytes + encoded > maximumBytes) break
        result.insert(0, characters)
        bytes += encoded
        index -= characters.size
    }
    return result.toString()
}
