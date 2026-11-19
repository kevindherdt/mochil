package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class WikiModelsTest {
    @Test fun `collapsed folders hide descendants without hiding siblings`() {
        val entries = listOf(
            WikiEntry("folder", path = "Books"),
            WikiEntry("folder", path = "Books/Fantasy"),
            WikiEntry("page", "1", "Books/Fantasy/Plan.md", "Plan", "md"),
            WikiEntry("page", "2", "Home.md", "Home", "md"),
        )
        assertEquals(listOf("Books", "Books/Fantasy", "Home.md"), WikiTreeIndex(entries).visible(setOf("Books")).map(WikiEntry::path))
        assertEquals(4, WikiTreeIndex(entries).visible(setOf("Books", "Books/Fantasy")).size)
    }

    @Test fun `page paths stay Obsidian compatible`() {
        assertEquals("Projects/Plan.md", normalizeWikiPagePath(" /Projects\\Plan "))
        assertEquals("Notes/Page.MD", normalizeWikiPagePath("Notes/Page.MD"))
    }

    @Test fun `search finds nested pages even while their folder is collapsed`() {
        val entries = listOf(
            WikiEntry("folder", path = "Projects"),
            WikiEntry("page", "1", "Projects/Roadmap.md", "Roadmap", "md"),
            WikiEntry("page", "2", "Home.md", "Home", "md"),
        )
        assertEquals(listOf("Projects/Roadmap.md"), WikiTreeIndex(entries).search("road", emptySet()).map(WikiEntry::path))
    }

    @Test fun `Obsidian links resolve by vault path or page name`() {
        val entries = listOf(WikiEntry("page", "1", "Projects/Roadmap.md", "Roadmap", "md"))
        assertEquals("1", findWikiLinkTarget(entries, "Projects/Roadmap")?.id)
        assertEquals("1", findWikiLinkTarget(entries, "roadmap")?.id)
        assertEquals("1", findWikiLinkTarget(entries, "Projects/Roadmap.md#Details")?.id)
        assertEquals(WikiLinkDestination("Projects/Roadmap", "Details"), parseWikiLinkDestination("Projects/Roadmap.md#Details"))
        assertEquals(WikiLinkDestination("", "Details"), parseWikiLinkDestination("#Details"))
    }

    @Test fun `missing simple Wiki links create beside the current page`() {
        assertEquals(
            WikiMissingLink("Next steps", "Projects/Next steps.md"),
            missingWikiLinkTarget("Projects/Roadmap.md", "Next steps#Later"),
        )
        assertEquals(
            WikiMissingLink("Index", "Reference/Index.md"),
            missingWikiLinkTarget("Projects/Roadmap.md", "Reference/Index"),
        )
    }

    @Test fun `explicit Wiki paths win over earlier duplicate page labels`() {
        val entries = listOf(
            WikiEntry("page", "wrong", "Archive/Index.md", "Index", "md"),
            WikiEntry("page", "right", "Projects/Index.MD", "Index", "md"),
        )
        assertEquals("right", findWikiLinkTarget(entries, "Projects/Index")?.id)
        assertEquals("right", findWikiLinkTarget(entries, "Projects/Index.MD#Notes")?.id)
        assertNull(findWikiLinkTarget(entries, "Missing/Index"))
    }

    @Test fun `Wiki page lookup does not resolve to an attachment`() {
        val entries = listOf(WikiEntry("file", "image", "Cover.png", "Cover", "png"))
        assertNull(findWikiLinkTarget(entries, "Cover"))
    }

    @Test fun `missing Wiki links reject traversal and empty targets`() {
        assertNull(missingWikiLinkTarget("Projects/Roadmap.md", "../Secrets"))
        assertNull(missingWikiLinkTarget("Projects/Roadmap.md", "#Details"))
    }

    @Test fun `vault tree keeps folders first and naturally orders numbered pages`() {
        val entries = listOf(
            WikiEntry("page", "10", "Page 10.md", "Page 10", "md"),
            WikiEntry("folder", path = "Z Notes"),
            WikiEntry("page", "2", "Page 2.md", "Page 2", "md"),
            WikiEntry("folder", path = "Books"),
            WikiEntry("page", "b10", "Books/Volume 10.md", "Volume 10", "md"),
            WikiEntry("page", "b2", "Books/Volume 2.md", "Volume 2", "md"),
        )
        assertEquals(
            listOf("Books", "Books/Volume 2.md", "Books/Volume 10.md", "Z Notes", "Page 2.md", "Page 10.md"),
            WikiTreeIndex(entries).visible(setOf("Books", "Z Notes")).map(WikiEntry::path),
        )
    }

    @Test fun `natural comparison remains bounded for very large numbers`() {
        assertEquals(-1, naturalWikiNameCompare("Page 99999999999999999999", "Page 100000000000000000000"))
    }

    @Test fun `attachments stay out of the page tree but remain resolvable for embeds`() {
        val entries = listOf(
            WikiEntry("folder", path = "Notes"),
            WikiEntry("page", "page", "Notes/Plan.md", "Plan", "md"),
            WikiEntry("page", "image", "Notes/Assets/map.png", "map", "png"),
            WikiEntry("page", "pdf", "Notes/reference.pdf", "reference", "pdf"),
        )
        assertEquals(listOf("Notes", "Notes/Plan.md"), WikiTreeIndex(entries).visible(setOf("Notes")).map(WikiEntry::path))
        assertEquals("image", WikiAssetIndex(entries).resolve("Notes/Plan.md", "Assets/map.png")?.id)
        assertEquals("image", WikiAssetIndex(entries).resolve("Elsewhere.md", "map.png")?.id)
    }

    @Test fun `image resolver rejects remote absolute and escaping targets`() {
        val entries = listOf(WikiEntry("page", "image", "image.png", "image", "png"))
        assertEquals(null, WikiAssetIndex(entries).resolve("Plan.md", "https://example.com/image.png"))
        assertEquals(null, WikiAssetIndex(entries).resolve("Plan.md", "../../image.png"))
        assertEquals(null, WikiAssetIndex(entries).resolve("Plan.md", "/image.png"))
    }

    @Test fun `image index prefers relative image over earlier root candidate`() {
        val rootFirst = WikiEntry("page", "root", "map.png", "map", "png")
        val relativeLater = WikiEntry("page", "relative", "Notes/map.png", "map", "png")
        assertEquals("relative", WikiAssetIndex(listOf(rootFirst, relativeLater)).resolve("Notes/Page.md", "map.png")?.id)
    }

    @Test fun `late backlinks cannot replace the page opened afterward`() {
        val current = WikiState(libraryId = "vault", page = WikiPage("new", "New", "", "revision", true))
        val stale = listOf(WikiEntry("page", "source", "Source.md", "Source", "md"))
        assertEquals(current, current.withBacklinksIfCurrent("vault", "old", stale))
        assertEquals(stale, current.withBacklinksIfCurrent("vault", "new", stale).backlinks)
        assertEquals(current, current.withBacklinksIfCurrent("another-vault", "new", stale))
    }

    @Test fun `offline page applies only to the selected vault path and stays read only`() {
        val current = WikiState(libraryId = "vault", selectedPath = "Notes/Plan.md", loading = true)
        val cached = WikiOfflinePage("Notes/Plan.md", WikiPage("page", "Plan", "cached", "r1", false), 10)
        val applied = current.withOfflinePageIfCurrent("vault", "Notes/Plan.md", cached)
        assertEquals("cached", applied.page?.content)
        assertEquals(true, applied.offlineCopy)
        assertEquals(current, current.withOfflinePageIfCurrent("other", "Notes/Plan.md", cached))
        assertEquals(current, current.withOfflinePageIfCurrent("vault", "Other.md", cached))
    }

    @Test fun `failed remote search falls back to local filtering while offline`() {
        val offline = WikiState(
            libraryId = "vault",
            searchQuery = "plan",
            searchResults = listOf(WikiEntry("page", "remote", "Remote.md", "Remote", "md")),
            searching = true,
            offlineCopy = true,
        ).withWikiSearchFailure("vault", "plan", "network unavailable")
        assertEquals("", offline.searchQuery)
        assertEquals(emptyList<WikiEntry>(), offline.searchResults)
        assertEquals(false, offline.searching)
        assertNull(offline.error)
    }

    @Test fun `failed remote search keeps an error online and ignores stale responses`() {
        val current = WikiState(libraryId = "vault", searchQuery = "plan", searching = true)
        assertEquals("network unavailable", current.withWikiSearchFailure("vault", "plan", "network unavailable").error)
        assertEquals(current, current.withWikiSearchFailure("vault", "other", "stale"))
        assertEquals(current, current.withWikiSearchFailure("other", "plan", "stale"))
    }
}
