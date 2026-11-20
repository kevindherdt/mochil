package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Test

class WikiTreeIndexTest {
    @Test
    fun nestedVisibilityRequiresEveryAncestorAndSurvivesRepeatedToggles() {
        val entries = listOf(
            WikiEntry("folder", path = "A"),
            WikiEntry("folder", path = "A/B"),
            WikiEntry("page", "deep", "A/B/Page.md", "Page", "md"),
            WikiEntry("page", "root", "Root.md", "Root", "md"),
        )
        val index = WikiTreeIndex(entries)
        repeat(20) {
            assertEquals(listOf("A", "Root.md"), index.visible(setOf("A/B")).map(WikiEntry::path))
            assertEquals(listOf("A", "A/B", "Root.md"), index.visible(setOf("A")).map(WikiEntry::path))
            assertEquals(listOf("A", "A/B", "A/B/Page.md", "Root.md"), index.visible(setOf("A", "A/B")).map(WikiEntry::path))
            assertEquals(listOf("A", "Root.md"), index.visible(emptySet()).map(WikiEntry::path))
        }
        assertEquals(listOf("deep"), index.search("page", emptySet()).map(WikiEntry::id))
    }

    private val entries = listOf(
        WikiEntry("page", "root", "10 Root.md", "10 Root", "md"),
        WikiEntry("folder", path = "Projects"),
        WikiEntry("page", "later", "Projects/10 Later.md", "10 Later", "md"),
        WikiEntry("page", "first", "Projects/2 First.md", "2 First", "md"),
        WikiEntry("file", "image", "Projects/cover.png", format = "png"),
    )

    @Test
    fun visibilityUsesThePreorderedTreeWithoutExposingAttachments() {
        val index = WikiTreeIndex(entries)

        assertEquals(listOf("Projects", "10 Root.md"), index.visible(emptySet()).map(WikiEntry::path))
        assertEquals(
            listOf("Projects", "Projects/2 First.md", "Projects/10 Later.md", "10 Root.md"),
            index.visible(setOf("Projects")).map(WikiEntry::path),
        )
    }

    @Test
    fun searchFindsCollapsedPagesWithoutReturningFolders() {
        assertEquals(listOf("Projects/10 Later.md"), WikiTreeIndex(entries).search("later", emptySet()).map(WikiEntry::path))
    }

    @Test
    fun repeatedSearchRetainsPathOrderAndBlankQueryRestoresTreeOrder() {
        val pages = entries + listOf(
            WikiEntry("page", "unicode", "Projects/Ä Note.md", "Café", "md"),
            WikiEntry("page", "upper", "Projects/NOTE.md", "Same", "md"),
            WikiEntry("page", "lower", "Projects/note.md", "Same", "md"),
        )
        val index = WikiTreeIndex(pages)
        val originalOrder = orderedWikiEntries(wikiPageTreeEntries(pages))
        repeat(10) {
            for (query in listOf("Projects", "note", "CAFÉ", "Same", "missing", "  later ")) {
                val needle = query.trim()
                val expected = originalOrder.filterNot(WikiEntry::isFolder)
                    .filter { it.label.contains(needle, true) || it.path.contains(needle, true) }
                    .sortedBy { it.path.lowercase() }
                assertEquals(expected, index.search(query, emptySet()))
            }
            assertEquals(index.visible(setOf("Projects")), index.search("  ", setOf("Projects")))
        }
        // A new tree must never reuse the previous vault snapshot's search cache.
        assertEquals(emptyList<WikiEntry>(), WikiTreeIndex(emptyList()).search("Projects", emptySet()))
    }
}
