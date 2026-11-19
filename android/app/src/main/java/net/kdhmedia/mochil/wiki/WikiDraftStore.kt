package net.kdhmedia.mochil.wiki

import android.content.Context
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.EncryptedPayloadStore
import org.json.JSONObject

internal data class WikiDraft(
    val libraryId: String,
    val pageId: String,
    val revision: String,
    val content: String,
    val updatedAt: Long,
)

internal fun WikiDraft?.recoverFor(libraryId: String, page: WikiPage): WikiDraft? = this?.takeIf {
    it.libraryId == libraryId && it.pageId == page.id && it.content != page.content
}

/** Only an app-start selection may claim the encrypted draft left for that exact page. */
internal fun WikiDraft?.retainForInitialSelection(currentPage: WikiPage?, libraryId: String, pageId: String): WikiDraft? = this?.takeIf {
    currentPage == null && it.libraryId == libraryId && it.pageId == pageId
}

internal fun shouldPersistWikiDraft(content: String, savedContent: String): Boolean = content != savedContent

/** Keeps only the active edit, encrypted at rest and bounded to one wiki page. */
internal class WikiDraftStore(context: Context) {
    private val encrypted = EncryptedPayloadStore(context, "wiki_active_draft", "mochil_wiki_active_draft")

    fun beginWrite(): Any = writes.begin()

    fun save(ownerId: String, draft: WikiDraft, ticket: Any) = writes.write(ticket) {
        require(ownerId.isNotBlank() && draft.isValidRecoveryDraft()) { "Invalid Wiki recovery draft" }
        val persisted = encrypted.save(JSONObject()
            .put("owner", wikiOwnerIdentity(ownerId))
            .put("library_id", draft.libraryId)
            .put("page_id", draft.pageId)
            .put("revision", draft.revision)
            .put("content", draft.content)
            .put("updated_at", draft.updatedAt)
            .toString().encodeToByteArray())
        if (!persisted) throw java.io.IOException("Could not persist Wiki recovery draft")
    }

    fun load(ownerId: String): WikiDraft? = writes.readVisible { encrypted.load()?.let { bytes -> runCatching {
        val value = JSONObject(bytes.decodeToString())
        if (value.optString("owner") != wikiOwnerIdentity(ownerId)) {
            encrypted.clear()
            return@runCatching null
        }
        decodeWikiDraft(value)
    }.getOrNull() } }

    fun clear() {
        writes.clearLater(dispatch = { operation ->
            clearScope.launch {
                try { operation() }
                catch (_: Exception) { Log.w("WikiDraftStore", "Could not clear the stored recovery draft") }
            }
        }) {
            if (!encrypted.clear()) throw java.io.IOException("Could not clear Wiki recovery draft")
        }
    }

    private companion object {
        // All instances share one encrypted preference file.
        val writes = WikiDraftWriteGate()
        // Explicit discard must finish even when its screen/ViewModel is destroyed.
        val clearScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    }
}

/** Recovery must be exact: invalid stored drafts are never silently shortened. */
internal fun decodeWikiDraft(value: JSONObject): WikiDraft {
    val draft = WikiDraft(
        value.getString("library_id"),
        value.getString("page_id"),
        value.optString("revision"),
        value.getString("content"),
        value.optLong("updated_at"),
    )
    require(draft.isValidRecoveryDraft()) { "Invalid Wiki recovery draft" }
    return draft
}

internal fun WikiDraft.isValidRecoveryDraft(): Boolean =
    libraryId.isNotBlank() && pageId.isNotBlank() && acceptsWikiDraft(content)

internal fun wikiOwnerIdentity(ownerId: String): String = java.security.MessageDigest.getInstance("SHA-256")
    .digest(ownerId.trim().encodeToByteArray()).take(12).joinToString("") { "%02x".format(it) }
