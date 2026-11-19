package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiPageLinksTest {
    private fun page(path: String) = WikiEntry("page", path, path, path.substringAfterLast('/').removeSuffix(".md"), "md")

    @Test fun `relative local and root matching is independent of entry order`() {
        val entries = listOf(page("Note.md"), page("A/Note.md"), page("B/Unique.md"))
        for (ordered in listOf(entries, entries.reversed())) {
            assertEquals("A/Note.md", findWikiLinkTarget(ordered, "Note", "A/Start.md")?.id)
            assertEquals("Note.md", findWikiLinkTarget(ordered, "../Note", "A/Start.md")?.id)
            assertEquals("B/Unique.md", findWikiLinkTarget(ordered, "B/Unique", "A/Start.md")?.id)
            assertNull(findWikiLinkTarget(ordered, "./Unique", "A/Start.md"))
            assertNull(findWikiLinkTarget(ordered, "../../Note", "A/Start.md"))
        }
    }

    @Test fun `ambiguous names are not chosen while exact case remains usable`() {
        val entries = listOf(page("A/Note.md"), page("B/Note.md"), page("Case.md"), page("case.md"))
        assertNull(findWikiLinkTarget(entries, "Note"))
        assertNull(findWikiLinkTarget(entries, "CASE"))
        assertEquals("Case.md", findWikiLinkTarget(entries, "Case")?.id)
        assertEquals("case.md", findWikiLinkTarget(entries, "case")?.id)
        for (target in listOf("/Case", "data:Case", "bad\u0000", "bad\u007f")) assertNull(findWikiLinkTarget(entries, target))
    }

    @Test fun `repeated entry identity is not a second candidate in a large catalogue`() {
        val target = page("Notes/Target.md")
        val entries = (0 until 10_000).map { page("Other/Page$it.md") } + target + target.copy()
        assertEquals(target, findWikiLinkTarget(entries, "Target", "Notes/Start.md"))
        assertEquals(target, findWikiLinkTarget(entries, "target", "Notes/Start.md"))
        assertEquals(target, findWikiLinkTarget(entries, "Target"))
        assertNull(findWikiLinkTarget(entries + target.copy(id = "different"), "Notes/Target"))
    }
}
