package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiCloseStateTest {
    @Test fun `closing an in flight page ends loading and clears page scoped controls`() {
        val state = WikiState(libraryId = "vault", loading = true, selectedPath = "Note.md",
            saving = true, editing = true, draft = "draft", draftRecovered = true,
            navigationHeading = "Heading", missingLink = WikiMissingLink("Missing", "Missing.md"),
            attachmentStatus = "Downloading", error = "Previous failure")
        val closed = state.withClosedWikiPage()
        assertFalse(closed.loading)
        assertFalse(closed.saving)
        assertFalse(closed.editing)
        assertFalse(closed.draftRecovered)
        assertEquals("", closed.selectedPath)
        assertEquals("", closed.draft)
        assertEquals("", closed.navigationHeading)
        assertNull(closed.missingLink)
        assertNull(closed.attachmentStatus)
        assertNull(closed.error)
    }

    @Test fun `closing preserves vault tree expansion search and independent offline operation`() {
        val entry = WikiEntry("page", "note", "Folder/Note.md", format = "md")
        val state = WikiState(libraryId = "vault", entries = listOf(entry), expanded = setOf("Folder"),
            page = WikiPage("note", "Note", "Content", "revision", true), backlinks = listOf(entry),
            searchQuery = "note", searchResults = listOf(entry), searching = true,
            offlineCopy = true, offlineSaving = true, offlineSaved = 4, offlineTotal = 10,
            offlineAssetUrls = mapOf("image" to "file:///private/image"))
        val closed = state.withClosedWikiPage()
        assertNull(closed.page)
        assertTrue(closed.backlinks.isEmpty())
        assertEquals(state, closed.copy(page = state.page, backlinks = state.backlinks))
        assertEquals(closed, closed.withClosedWikiPage())
    }
}
