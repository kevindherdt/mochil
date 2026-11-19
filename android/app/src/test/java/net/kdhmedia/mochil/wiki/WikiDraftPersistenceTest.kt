package net.kdhmedia.mochil.wiki

import java.io.IOException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.runBlocking
import org.junit.Assert.*
import org.junit.Test

class WikiDraftPersistenceTest {
    private val draft = WikiState(libraryId = "vault", selectedPath = "Note.md",
        page = WikiPage("note", "Note", "original", "r1", true), draft = "my edits", editing = true)

    @Test fun `storage failure is reported without discarding draft`() = runBlocking {
        assertFalse(persistWikiDraft { throw IOException("disk unavailable") })
        val result = draft.withDraftPersistenceFailure(draft)
        assertEquals(draft.draft, result.draft)
        assertEquals(draft.page, result.page)
        assertTrue(result.editing)
        assertNotNull(result.error)
    }

    @Test fun `successful persistence is acknowledged`() = runBlocking {
        assertTrue(persistWikiDraft {})
    }

    @Test fun `cancellation remains cancellation`() = runBlocking {
        val cancellation = CancellationException("new edit")
        try {
            persistWikiDraft { throw cancellation }
            fail("cancellation swallowed")
        } catch (actual: CancellationException) {
            assertSame(cancellation, actual)
        }
    }

    @Test fun `old failure cannot affect newer edit or another page`() {
        for (current in listOf(draft.copy(draft = "newer"), draft.copy(selectedPath = "Other.md"),
            draft.copy(libraryId = "other"), draft.copy(editing = false))) {
            assertSame(current, current.withDraftPersistenceFailure(draft))
        }
    }
}
