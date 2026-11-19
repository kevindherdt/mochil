package net.kdhmedia.mochil.reader.epub

import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class EpubAnnotationSessionTest {
    @Test fun sessionOwnsObservableLocalStateAcrossEditsAndDeletion() = runBlocking {
        val annotation = SyncedAnnotation("mark", "highlight", quote = "Selected text", revision = 4)
        val store = SessionAnnotationStore(LocalAnnotationState(items = mapOf(annotation.id to annotation)))
        val session = EpubAnnotationSession(EpubAnnotationController(store, "book"))

        session.load()
        assertTrue(session.editNote("mark", "Remember this"))
        assertEquals("Remember this", session.state.items.getValue("mark").note)
        assertTrue(session.delete("mark"))
        assertTrue(session.state.items.getValue("mark").deleted)
        assertEquals(2, store.saves)
    }

    @Test fun remoteFeaturesRequireACompleteExplicitBinding() {
        val session = EpubAnnotationSession(EpubAnnotationController(SessionAnnotationStore(), "book"))
        val credential = DeviceCredential("https://media.example", "device-one", "token")

        session.bindRemote(credential, "", "book")
        assertFalse(session.notesAvailable)
        assertEquals("", session.deviceId)

        session.bindRemote(credential, "library", "book")
        assertTrue(session.notesAvailable)
        assertEquals("device-one", session.deviceId)

        session.bindRemote(null, "library", "book")
        assertFalse(session.notesAvailable)
    }
}

private class SessionAnnotationStore(
    initial: LocalAnnotationState = LocalAnnotationState(),
) : AnnotationLocalStore {
    var state = initial
    var saves = 0

    override suspend fun load(publicationKey: String) = state
    override suspend fun save(publicationKey: String, state: LocalAnnotationState) {
        this.state = state
        saves++
    }
    override fun needsStableLocatorRepair(publicationKey: String) = false
    override fun markStableLocatorRepair(publicationKey: String) = Unit
}
