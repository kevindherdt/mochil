package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiMoveStateTest {
    private val page = WikiPage("old-id", "Note", "saved", "r1", true)
    private val submitted = WikiState(libraryId = "vault", page = page, selectedPath = "Note.md", draft = "saved", editing = true, saving = true)
    private val moved = WikiEntry("file", "new-id", "Folder/Note.md")
    private val result = WikiMutationResult(listOf(moved), moved, page.copy(id = moved.id))

    @Test fun `new text and intentional deletion follow moved page`() {
        for (draft in listOf("new text", "")) {
            val state = submitted.copy(draft = draft).withMovedWikiPage(submitted, result)
            assertEquals(draft, state.draft)
            assertEquals(moved.id, state.page?.id)
            assertEquals(moved.path, state.selectedPath)
            assertEquals("r1", state.page?.revision)
            assertEquals("saved", state.page?.content)
            assertTrue(state.editing)
            assertFalse(state.saving)
            assertTrue("Folder" in state.expanded)
        }
    }

    @Test fun `unchanged draft finishes moving without entering edit mode`() {
        val state = submitted.withMovedWikiPage(submitted, result)
        assertEquals("saved", state.draft)
        assertFalse(state.editing)
    }

    @Test fun `late result cannot replace another selection`() {
        for (state in listOf(submitted.copy(libraryId = "other"), submitted.copy(selectedPath = "Else.md"), submitted.copy(page = page.copy(id = "other")))) {
            assertSame(state, state.withMovedWikiPage(submitted, result))
        }
    }
}
