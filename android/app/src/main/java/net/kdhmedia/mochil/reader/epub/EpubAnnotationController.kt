package net.kdhmedia.mochil.reader.epub

import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import net.kdhmedia.mochil.auth.DeviceCredential
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

internal data class AnnotationConflictResolution(val changed: Boolean, val synchronize: Boolean)
internal data class AnnotationNotesExport(val state: LocalAnnotationState, val page: String)

internal class EpubAnnotationController(
    private val store: AnnotationLocalStore,
    private val publicationKey: String,
    private val client: AnnotationClient = AnnotationClient(),
) {
    private val mutex = Mutex()
    var state = LocalAnnotationState()
        private set

    suspend fun load(): LocalAnnotationState = mutex.withLock {
        state = store.load(publicationKey)
        state
    }

    suspend fun repairStableLocators(publication: Publication): LocalAnnotationState = mutex.withLock {
        if (!store.needsStableLocatorRepair(publicationKey)) return@withLock state
        state = resolveTextAnchors(publication, state, replaceExisting = true)
        store.save(publicationKey, state)
        store.markStableLocatorRepair(publicationKey)
        state
    }

    suspend fun createHighlight(locator: Locator, deviceId: String, spineIndex: Int): Boolean = mutex.withLock {
        update(state.createHighlight(locator, deviceId, spineIndex))
    }

    suspend fun delete(id: String, deviceId: String): Boolean = mutex.withLock {
        update(state.delete(id, deviceId))
    }

    suspend fun editNote(id: String, note: String, deviceId: String): Boolean = mutex.withLock {
        update(state.editNote(id, note, deviceId))
    }

    suspend fun resolveConflict(id: String, keepLocal: Boolean): AnnotationConflictResolution = mutex.withLock {
        val changed = update(if (keepLocal) state.keepLocal(id) else state.useRemote(id))
        AnnotationConflictResolution(changed, changed && keepLocal)
    }

    suspend fun synchronize(publication: Publication, credential: DeviceCredential, libraryId: String, bookId: String): LocalAnnotationState = mutex.withLock {
        synchronizeLocked(publication, credential, libraryId, bookId)
    }

    suspend fun exportToNotes(publication: Publication, credential: DeviceCredential, libraryId: String, bookId: String): AnnotationNotesExport = mutex.withLock {
        val synchronized = synchronizeLocked(publication, credential, libraryId, bookId)
        if (synchronized.conflicts.isNotEmpty()) throw AnnotationNotesException(409)
        AnnotationNotesExport(synchronized, AnnotationNotesClient().create(credential, libraryId, bookId))
    }

    private suspend fun synchronizeLocked(publication: Publication, credential: DeviceCredential, libraryId: String, bookId: String): LocalAnnotationState {
        state = state.mergePull(client.changes(credential, libraryId, bookId, state.serverRevision))
        state = resolveTextAnchors(publication, state)
        store.save(publicationKey, state)
        val pending = state.pendingChanges().filterNot { it.annotation.id in state.conflicts }
        if (pending.isNotEmpty()) state = state.mergeApply(client.apply(credential, libraryId, bookId, pending))
        store.save(publicationKey, state)
        return state
    }

    private suspend fun update(updated: LocalAnnotationState): Boolean {
        if (updated == state) return false
        state = updated
        store.save(publicationKey, updated)
        return true
    }

    private suspend fun resolveTextAnchors(publication: Publication, source: LocalAnnotationState, replaceExisting: Boolean = false): LocalAnnotationState {
        var resolved = source
        source.visible.asSequence()
            .filter { (replaceExisting || it.readiumLocator() == null) && !it.anchor?.exact.isNullOrBlank() }
            .take(MAX_ANCHORS_RESOLVED_PER_SYNC)
            .forEach { annotation ->
                val anchor = annotation.anchor ?: return@forEach
                val expectedHref = publication.readingOrder.getOrNull(anchor.spineIndex)?.url()
                val matches = runCatching { EpubSearch.find(publication, anchor.exact, MAX_ANCHOR_MATCHES) }.getOrDefault(emptyList())
                val best = matches.maxByOrNull { locator ->
                    var score = if (expectedHref != null && locator.href == expectedHref) 100 else 0
                    if (anchor.prefix.isNotBlank() && locator.text.before.orEmpty().endsWith(anchor.prefix)) score += 20
                    if (anchor.suffix.isNotBlank() && locator.text.after.orEmpty().startsWith(anchor.suffix)) score += 20
                    score
                }
                if (best != null && (expectedHref == null || best.href == expectedHref || matches.size == 1)) {
                    val encoded = best.toJSON().toString()
                    resolved = if (replaceExisting) resolved.withResolvedReadiumLocator(annotation.id, encoded) else resolved.withDerivedReadiumLocator(annotation.id, encoded)
                }
            }
        return resolved
    }

    private companion object {
        const val MAX_ANCHORS_RESOLVED_PER_SYNC = 50
        const val MAX_ANCHOR_MATCHES = 50
    }
}
