package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiLoadStateTest {
    private val page = WikiPage("note", "Note", "cached", "r1", true)
    private val submitted = WikiState(libraryId = "vault", selectedPath = "Note.md",
        page = page, draft = page.content, offlineCopy = true, loading = true)
    private val loaded = submitted.copy(page = page.copy(content = "remote", revision = "r2"),
        draft = "remote", offlineCopy = false, loading = false)

    @Test fun `cached page edits survive remote loading with original save revision`() {
        val current = submitted.copy(editing = true, draft = "my changes")
        val result = current.withWikiLoadedState(submitted, loaded)
        assertEquals(current.page, result.page)
        assertEquals("my changes", result.draft)
        assertTrue(result.editing)
        assertTrue(result.offlineCopy)
        assertEquals(false, result.loading)
    }

    @Test fun `completed save is not rolled back by old load`() {
        val current = submitted.copy(page = page.copy(content = "saved", revision = "r3"), draft = "saved", offlineCopy = false)
        val result = current.withWikiLoadedState(submitted, loaded)
        assertEquals(current.page, result.page)
        assertEquals("saved", result.draft)
    }

    @Test fun `save in flight and recovered draft retain their state`() {
        val current = submitted.copy(editing = true, saving = true, draftRecovered = true, draft = "recovered")
        val result = current.withWikiLoadedState(submitted, loaded)
        assertTrue(result.saving)
        assertTrue(result.draftRecovered)
        assertEquals("recovered", result.draft)
    }

    @Test fun `unmodified page accepts remote copy`() {
        assertSame(loaded, submitted.withWikiLoadedState(submitted, loaded))
    }

    @Test fun `another selection ignores old load`() {
        val current = submitted.copy(selectedPath = "Other.md")
        assertSame(current, current.withWikiLoadedState(submitted, loaded))
    }
}
