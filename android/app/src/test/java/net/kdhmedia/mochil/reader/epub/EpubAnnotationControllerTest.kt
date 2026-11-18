package net.kdhmedia.mochil.reader.epub

import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EpubAnnotationControllerTest {
    @Test fun noteEditPersistsThroughTheControllerBoundary() = runBlocking {
        val annotation = SyncedAnnotation("mark", "highlight", quote = "Selected text", revision = 3)
        val store = FakeAnnotationStore(LocalAnnotationState(items = mapOf(annotation.id to annotation)))
        val controller = EpubAnnotationController(store, "book")
        controller.load()

        assertTrue(controller.editNote("mark", "My note", "device"))
        assertEquals("My note", store.state.items.getValue("mark").note)
        assertEquals(3L, store.state.pendingBaseRevisions["mark"])
    }

    @Test fun unchangedEditDoesNotPerformAnotherDurableWrite() = runBlocking {
        val annotation = SyncedAnnotation("mark", "highlight", note = "Same")
        val store = FakeAnnotationStore(LocalAnnotationState(items = mapOf(annotation.id to annotation)))
        val controller = EpubAnnotationController(store, "book")
        controller.load()

        assertFalse(controller.editNote("mark", "Same", "device"))
        assertEquals(0, store.saves)
    }

    @Test fun deletePersistsATombstoneInsteadOfDroppingIdentity() = runBlocking {
        val annotation = SyncedAnnotation("mark", "highlight", quote = "Selected text", revision = 7)
        val store = FakeAnnotationStore(LocalAnnotationState(items = mapOf(annotation.id to annotation)))
        val controller = EpubAnnotationController(store, "book")
        controller.load()

        assertTrue(controller.delete("mark", "device"))
        assertTrue(store.state.items.getValue("mark").deleted)
        assertEquals(7L, store.state.pendingBaseRevisions["mark"])
    }
}

private class FakeAnnotationStore(initial: LocalAnnotationState) : AnnotationLocalStore {
    var state = initial
    var saves = 0
    override suspend fun load(publicationKey: String) = state
    override suspend fun save(publicationKey: String, state: LocalAnnotationState) { this.state = state; saves++ }
    override fun needsStableLocatorRepair(publicationKey: String) = false
    override fun markStableLocatorRepair(publicationKey: String) = Unit
}
