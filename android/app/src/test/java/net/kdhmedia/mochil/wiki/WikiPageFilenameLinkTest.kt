package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiPageFilenameLinkTest {
    @Test fun `already parsed destinations retain a literal md filename suffix`() {
        val literal = WikiEntry("page", "literal", "Docs/Report.md.md", "Literal filename", "md")
        val other = WikiEntry("page", "other", "Docs/Report.md", "Other note", "md")
        val raw = "Report.md.md#Details|Label"
        val destination = parseWikiLinkDestination(raw)
        assertEquals("Report.md", destination.page)
        assertEquals("Details", destination.heading)
        assertEquals(literal, findWikiLinkTarget(listOf(other, literal), destination, "Docs/Start.md"))
        assertEquals(literal, findWikiLinkTarget(listOf(other, literal), raw, "Docs/Start.md"))
    }

    @Test fun `short links use filenames even when a note has a different display title`() {
        val note = WikiEntry("page", "note", "Projects/Roadmap.md", "Future plans", "md")
        assertEquals(note, findWikiLinkTarget(listOf(note), "Roadmap", "Journal/Today.md"))
        assertEquals(note, findWikiLinkTarget(listOf(note), "roadmap.md#Next", "Journal/Today.md"))
    }

    @Test fun `filename identity wins over another page display title`() {
        val note = WikiEntry("page", "note", "Projects/Roadmap.md", "Future plans", "md")
        val alias = WikiEntry("page", "alias", "Other/Unrelated.md", "Roadmap", "md")
        assertEquals(note, findWikiLinkTarget(listOf(alias, note), "Roadmap"))
    }

    @Test fun `ambiguous short filenames cannot be resolved by title or scan order`() {
        val first = WikiEntry("page", "one", "Projects/Index.md", "Index", "md")
        val second = WikiEntry("page", "two", "Other/Index.md", "Other title", "md")
        assertNull(findWikiLinkTarget(listOf(first, second), "Index"))
        assertNull(findWikiLinkTarget(listOf(second, first), "Index"))
        assertEquals(first, findWikiLinkTarget(listOf(first, second), "Index", "Projects/Today.md"))
    }

    @Test fun `exact filename case is preferred before folded fallback`() {
        val first = WikiEntry("page", "one", "Projects/Index.md", "One", "md")
        val second = WikiEntry("page", "two", "Other/index.md", "Two", "md")
        assertEquals(first, findWikiLinkTarget(listOf(second, first), "Index"))
        assertEquals(second, findWikiLinkTarget(listOf(first, second), "index"))
        assertNull(findWikiLinkTarget(listOf(first, second), "INDEX"))
    }
}
