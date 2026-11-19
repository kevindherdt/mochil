package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class AnnotationStateTest {
    @Test fun highlightSearchQueryNormalizesSelectionWhitespace() {
        assertEquals("One selected sentence.", stableHighlightQuery("  One\n selected\t sentence.  "))
    }

    @Test fun stableHighlightCandidatesPreferTheSelectedResourceAndAvoidAmbiguousFallbacks() {
        assertEquals(listOf(1, 2), stableSelectionCandidateIndexes("chapter.xhtml", listOf("other.xhtml", "chapter.xhtml", "chapter.xhtml")))
        assertEquals(listOf(0), stableSelectionCandidateIndexes("chapter.xhtml", listOf("renamed.xhtml")))
        assertTrue(stableSelectionCandidateIndexes("chapter.xhtml", listOf("one.xhtml", "two.xhtml")).isEmpty())
    }

    @Test fun equivalentReadingPositionsIgnoreTinyRendererDifferences() {
        assertTrue(readingPositionValuesEquivalent(0.401, "chapter.xhtml", 0.101, 0.4025, "chapter.xhtml", 0.1025))
    }

    @Test fun visibleAnnotationsCollapseCrossDeviceIdentityDuplicates() {
        val anchor = ServerTextAnchor("Before", "Same selected text", "After", 3, 0.4)
        val android = SyncedAnnotation("android-id", "highlight", quote = anchor.exact, updatedAt = 20, anchor = anchor)
        val koreader = android.copy(id = "koreader-id", updatedAt = 10)
        val state = LocalAnnotationState(items = mapOf(android.id to android, koreader.id to koreader))
        assertEquals(listOf("android-id"), state.visible.map { it.id })
    }

    @Test fun selectedHighlightMatchesOnlyTheSamePublicationRange() {
        assertTrue(highlightRangesMatch("chapter.xhtml", 0.25, "Some context", " follows", "chapter.xhtml", 0.251, "Some context", " follows"))
        assertTrue(!highlightRangesMatch("chapter.xhtml", 0.25, "Some context", " follows", "other.xhtml", 0.251, "Some context", " follows"))
        assertTrue(!highlightRangesMatch("chapter.xhtml", 0.25, "Some context", " follows", "chapter.xhtml", 0.8, "Some context", " follows"))
    }

    @Test fun derivedReadiumLocatorIsLocalOnlyAndMakesRemoteHighlightRenderable() {
        val remote = SyncedAnnotation("mark", "highlight", quote = "A useful sentence", revision = 4)
        val state = LocalAnnotationState(serverRevision = 4, items = mapOf(remote.id to remote))
        val encoded = "{\"href\":\"chapter-2.xhtml\",\"type\":\"application/xhtml+xml\"}"
        val resolved = state.withDerivedReadiumLocator(remote.id, encoded)
        assertEquals(encoded, resolved.items.getValue(remote.id).locators["readium"])
        assertTrue(resolved.pendingChanges().isEmpty())
        assertEquals(4L, resolved.serverRevision)
    }

    @Test fun stableLocatorRepairReplacesAnExistingFragileLocatorWithoutCreatingASyncEdit() {
        val remote = SyncedAnnotation("mark", "highlight", quote = "A useful sentence", revision = 4, locators = mapOf("readium" to "old"))
        val state = LocalAnnotationState(serverRevision = 4, items = mapOf(remote.id to remote))
        val resolved = state.withResolvedReadiumLocator(remote.id, "stable")
        assertEquals("stable", resolved.items.getValue(remote.id).locators["readium"])
        assertTrue(resolved.pendingChanges().isEmpty())
    }

    @Test fun independentRemoteItemsMergeWhilePendingLocalEditIsProtected() {
        val local = SyncedAnnotation("shared", "note", note = "local", revision = 1)
        val state = LocalAnnotationState(items = mapOf(local.id to local), pendingBaseRevisions = mapOf(local.id to 1))
        val remoteShared = local.copy(note = "remote", revision = 2)
        val independent = SyncedAnnotation("other", "bookmark", revision = 3)
        val merged = state.mergePull(AnnotationSyncResult(3, listOf(remoteShared, independent), emptyList()))
        assertEquals("local", merged.items.getValue("shared").note)
        assertEquals(independent, merged.items["other"])
        assertTrue("shared" in merged.conflicts)
    }

    @Test fun successfulApplyClearsOnlyAppliedPendingItems() {
        val first = SyncedAnnotation("first", "note", note = "one")
        val second = SyncedAnnotation("second", "note", note = "two")
        val state = LocalAnnotationState(items = mapOf(first.id to first, second.id to second), pendingBaseRevisions = mapOf(first.id to 0, second.id to 0))
        val saved = first.copy(revision = 1, updatedAt = 10)
        val merged = state.mergeApply(AnnotationSyncResult(1, listOf(saved), emptyList()))
        assertEquals(mapOf("second" to 0L), merged.pendingBaseRevisions)
        assertEquals(1, merged.items.getValue("first").revision)
    }

    @Test fun deleteCreatesRevisionedTombstone() {
        val existing = SyncedAnnotation("mark", "highlight", quote = "text", revision = 7)
        val deleted = LocalAnnotationState(items = mapOf(existing.id to existing)).delete("mark", "android", 20)
        assertTrue(deleted.items.getValue("mark").deleted)
        assertEquals(7L, deleted.pendingBaseRevisions["mark"])
        assertTrue(deleted.visible.isEmpty())
    }

    @Test fun conflictsCanResolveToEitherSide() {
        val local = SyncedAnnotation("mark", "note", note = "mine", revision = 1)
        val remote = local.copy(note = "other", revision = 2)
        val conflict = AnnotationConflict(local, remote)
        val state = LocalAnnotationState(items = mapOf(local.id to local), pendingBaseRevisions = mapOf(local.id to 1), conflicts = mapOf(local.id to conflict))
        val mine = state.keepLocal(local.id)
        assertEquals("mine", mine.items.getValue(local.id).note)
        assertEquals(2L, mine.pendingBaseRevisions[local.id])
        assertTrue(mine.conflicts.isEmpty())
        val other = state.useRemote(local.id)
        assertEquals("other", other.items.getValue(local.id).note)
        assertTrue(other.pendingBaseRevisions.isEmpty())
        assertTrue(other.conflicts.isEmpty())
    }

    @Test fun editingNotePreservesOriginalBaseRevisionAndCanClearNote() {
        val existing = SyncedAnnotation("mark", "highlight", quote = "text", note = "old", revision = 7)
        val edited = LocalAnnotationState(items = mapOf(existing.id to existing)).editNote("mark", " new note ", "android", 20)
        assertEquals("new note", edited.items.getValue("mark").note)
        assertEquals(7L, edited.pendingBaseRevisions["mark"])
        val editedAgain = edited.editNote("mark", "", "android", 30)
        assertEquals("", editedAgain.items.getValue("mark").note)
        assertEquals(7L, editedAgain.pendingBaseRevisions["mark"])
    }

    @Test fun editingConflictedOrDeletedAnnotationIsRefused() {
        val existing = SyncedAnnotation("mark", "highlight", quote = "text", revision = 1)
        val conflict = AnnotationConflict(existing, existing.copy(note = "remote", revision = 2))
        val conflicted = LocalAnnotationState(items = mapOf(existing.id to existing), conflicts = mapOf(existing.id to conflict))
        assertEquals(conflicted, conflicted.editNote("mark", "local", "android"))
        assertEquals(conflicted, conflicted.delete("mark", "android"))
        val deleted = LocalAnnotationState(items = mapOf(existing.id to existing.copy(deleted = true)))
        assertEquals(deleted, deleted.editNote("mark", "local", "android"))
    }

    @Test fun noteLimitIsMeasuredInUtf8BytesWithoutSplittingCharacters() {
        val existing = SyncedAnnotation("mark", "highlight", quote = "text", revision = 1)
        val oversized = "🙂".repeat(5_000)
        val edited = LocalAnnotationState(items = mapOf(existing.id to existing)).editNote("mark", oversized, "android")
        val note = edited.items.getValue("mark").note
        assertTrue(note.encodeToByteArray().size <= LocalAnnotationState.MAX_NOTE_BYTES)
        assertTrue(note.all { !Character.isSurrogate(it) } || note.toCharArray().count(Character::isSurrogate) % 2 == 0)
    }
}
