package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiMutationPolicyTest {
    private val page = WikiPage("note", "Note", "saved", "r1", true)

    @Test fun `unsaved text including intentional deletion blocks page replacement`() {
        for (draft in listOf("new text", "")) {
            val state = WikiState(page = page, draft = draft, editing = true)
            assertNotNull(state.pageMutationWarning())
            assertEquals(draft, state.draft)
        }
    }

    @Test fun `unchanged or non editing page does not block structural actions`() {
        assertNull(WikiState(page = page, draft = page.content, editing = true).pageMutationWarning())
        assertNull(WikiState(page = page, draft = page.content).pageMutationWarning())
        assertNull(WikiState().pageMutationWarning())
    }

    @Test fun `confirmed trash stops editing and rejects pending editor callbacks`() {
        val before = WikiState(page = page, draft = page.content, editing = true)
        val pending = before.withTrashStarted()
        assertTrue(pending.saving)
        assertFalse(pending.editing)
        assertFalse(pending.acceptsDraftChange())
        assertEquals(before.draft, pending.draft)
        assertEquals(before.page, pending.page)
    }

    @Test fun `ordinary save still accepts newer edits and finished mutations unlock changes`() {
        assertTrue(WikiState(editing = true, saving = true).acceptsDraftChange())
        assertTrue(WikiState(editing = false, saving = false).acceptsDraftChange())
    }
}
