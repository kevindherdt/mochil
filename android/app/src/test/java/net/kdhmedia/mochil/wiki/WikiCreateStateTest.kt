package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiCreateStateTest {
    private val page = WikiPage("old", "Old", "saved", "r1", true)
    private val submitted = WikiState(libraryId = "vault", page = page, selectedPath = "Old.md", draft = "saved", editing = true, saving = true)
    private val entry = WikiEntry("file", "new", "New.md")
    private val created = WikiMutationResult(listOf(entry), entry, page.copy(id = "new", content = "new"))

    @Test fun `late page creation adds entry without replacing newer text`() {
        val current = submitted.copy(draft = "typed while creating")
        val result = current.withCreatedWikiEntry(submitted, created, "New.md")
        assertEquals(current.draft, result.draft)
        assertEquals(current.page, result.page)
        assertEquals("Old.md", result.selectedPath)
        assertEquals(created.entries, result.entries)
        assertTrue(result.editing)
        assertFalse(result.saving)
    }

    @Test fun `folder creation keeps editor and its draft active`() {
        val current = submitted.copy(draft = "unsaved")
        val result = current.withCreatedWikiEntry(current, WikiMutationResult(emptyList(), null, null), "Folder")
        assertEquals("unsaved", result.draft)
        assertTrue(result.editing)
        assertEquals(page, result.page)
    }

    @Test fun `unchanged page creation opens the new page but stale identity cannot`() {
        val result = submitted.withCreatedWikiEntry(submitted, created, "New.md")
        assertEquals("new", result.page?.id)
        assertEquals("new", result.draft)
        val elsewhere = submitted.copy(libraryId = "elsewhere")
        assertSame(elsewhere, elsewhere.withCreatedWikiEntry(submitted, created, "New.md"))
    }
}
