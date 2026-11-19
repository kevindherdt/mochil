package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiRefreshStateTest {
    private val page = WikiPage("note", "Note", "original", "r1", true)
    private val initial = WikiState(libraryId = "vault", selectedPath = "Note.md", page = page, draft = page.content)
    private val result = WikiRefreshResult(emptyList(), true, page.copy(content = "remote", revision = "r2"))

    @Test fun `editing started during refresh retains text and original conflict baseline`() {
        val current = initial.copy(editing = true, draft = "my changes")
        val refreshed = current.withWikiRefresh(initial, result)
        assertEquals(current.page, refreshed.page)
        assertEquals(current.draft, refreshed.draft)
        assertTrue(refreshed.editing)
    }

    @Test fun `refresh cannot roll back a save completed meanwhile`() {
        val current = initial.copy(page = page.copy(content = "saved", revision = "r3"), draft = "saved")
        assertEquals(current.page, current.withWikiRefresh(initial, result).page)
    }

    @Test fun `remote deletion preserves editable draft and selection`() {
        val current = initial.copy(editing = true, draft = "my changes")
        val refreshed = current.withWikiRefresh(initial, result.copy(selectedStillExists = false, page = null))
        assertEquals(current.selectedPath, refreshed.selectedPath)
        assertEquals(current.draft, refreshed.draft)
        assertTrue(refreshed.error.orEmpty().contains("draft"))
    }

    @Test fun `unmodified page accepts fresh remote content`() {
        assertEquals(result.page, initial.withWikiRefresh(initial, result).page)
        assertEquals("remote", initial.withWikiRefresh(initial, result).draft)
    }

    @Test fun `another selection ignores the refresh`() {
        val current = initial.copy(selectedPath = "Other.md")
        assertSame(current, current.withWikiRefresh(initial, result))
    }
}
