package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiSaveStateTest {
    private val page = WikiPage("note", "Note", "old", "r1", true)
    private val submitted = WikiState(libraryId = "vault", page = page, selectedPath = "Note.md", draft = "submitted", editing = true, saving = true)
    private val saved = page.copy(content = "submitted", revision = "r2")

    @Test fun `newer text remains editable with the acknowledged revision`() {
        val current = submitted.copy(draft = "submitted plus more")
        val result = current.withSavedWikiDraft(submitted, saved)
        assertEquals("submitted plus more", result.draft)
        assertEquals("r2", result.page?.revision)
        assertEquals("submitted", result.page?.content)
        assertTrue(result.editing)
        assertFalse(result.saving)
    }

    @Test fun `unchanged submission finishes editing`() {
        val result = submitted.withSavedWikiDraft(submitted, saved)
        assertEquals(saved, result.page)
        assertEquals(saved.content, result.draft)
        assertFalse(result.editing)
        assertFalse(result.saving)
    }

    @Test fun `late save cannot alter another page or its saving state`() {
        for (current in listOf(submitted.copy(libraryId = "other"), submitted.copy(page = page.copy(id = "other")), submitted.copy(selectedPath = "Other.md"))) {
            assertSame(current, current.withSavedWikiDraft(submitted, saved))
        }
    }
}
