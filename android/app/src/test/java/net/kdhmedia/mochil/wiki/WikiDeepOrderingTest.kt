package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Test

class WikiDeepOrderingTest {
    @Test fun `deep hierarchy uses iterative traversal and preserves every descendant`() {
        val entries = (1..2000).map { depth -> WikiEntry("folder", path = List(depth) { "a" }.joinToString("/")) }
        val page = WikiEntry("page", "leaf", "${entries.last().path}/Note.md", "Note", "md")
        assertEquals(entries + page, orderedWikiEntries((entries + page).reversed()))
    }

    @Test fun `empty root entry cannot recursively reopen the root`() {
        val page = WikiEntry("page", "page", "Note.md", "Note", "md")
        assertEquals(listOf(page), orderedWikiEntries(listOf(WikiEntry("folder", path = ""), page)))
    }

    @Test fun `finishing a nested branch resumes its siblings and then root pages`() {
        val paths = listOf("Z.md", "B", "A", "A/2.md", "A/1", "A/1/Note.md", "B/Note.md")
        val entries = paths.map { WikiEntry(if (it.endsWith(".md")) "page" else "folder", path = it) }
        assertEquals(listOf("A", "A/1", "A/1/Note.md", "A/2.md", "B", "B/Note.md", "Z.md"),
            orderedWikiEntries(entries).map { it.path })
    }
}
