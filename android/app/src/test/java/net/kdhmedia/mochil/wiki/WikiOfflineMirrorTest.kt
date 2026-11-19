package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.util.concurrent.CancellationException

class WikiOfflineMirrorTest {
    @Test fun `partial refresh retains images referenced by retained pages`() {
        val entries = listOf(
            WikiEntry("page", "old", "Old.md", format = "md"),
            WikiEntry("page", "new", "New.md", format = "md"),
            WikiEntry("asset", "old-image", "old.png", format = "png"),
            WikiEntry("asset", "new-image", "new.png", format = "png"),
        )
        val oldPage = WikiOfflinePage("Old.md", WikiPage("old", "Old", "![[old.png]]", "r1", false), 1)
        val newPage = WikiOfflinePage("New.md", WikiPage("new", "New", "![[new.png]]", "r2", false), 2)
        val previous = WikiOfflineSnapshot("vault", entries, "Old.md", listOf(oldPage), 1)
        val incoming = WikiOfflineSnapshot("vault", entries, "New.md", listOf(newPage), 2)
        val merged = mergeWikiOfflineSnapshots(previous, incoming)
        val persisted = decodeWikiOfflineSnapshot(boundedWikiOfflinePayload(merged)!!)!!
        assertEquals(setOf("old-image", "new-image"),
            wikiOfflineAssetTargets(persisted.entries, persisted.pages).map { it.id }.toSet())
    }

    @Test fun `asset planning can cancel inside a long note`() {
        val content = (1..1000).joinToString("\n") { "Text $it" }
        val page = WikiOfflinePage("Note.md", WikiPage("note", "Note", content, "r", false), 1)
        var checks = 0
        var cancelled = false
        try {
            wikiOfflineAssetTargets(emptyList(), listOf(page)) {
                if (++checks == 10) throw CancellationException("Vault closed")
            }
        } catch (_: CancellationException) { cancelled = true }
        assertTrue(cancelled)
        assertEquals(10, checks)
    }

    @Test fun `zero asset capacity skips note parsing`() {
        val page = WikiOfflinePage("Note.md", WikiPage("note", "Note", "![[map.png]]", "r", false), 1)
        var checks = 0
        assertTrue(wikiOfflineAssetTargets(emptyList(), listOf(page), 0) { checks++ }.isEmpty())
        assertEquals(1, checks)
    }

    @Test fun `offline mirror capacity is bounded independently from tree size`() {
        val entries = List(MAX_OFFLINE_PAGES + 4) { index -> WikiEntry("page", "page-$index", "Page $index.md", "Page $index", "md") } +
            WikiEntry("asset", "image", "image.png", "image", "png")
        val selectedPath = "Page ${MAX_OFFLINE_PAGES + 3}.md"
        val targets = wikiOfflineMirrorTargets(entries, selectedPath)
        assertEquals(MAX_OFFLINE_PAGES, targets.size)
        assertTrue(targets.all { it.format == "md" })
        assertFalse(targets.any { it.id == "image" })
        assertEquals(selectedPath, targets.first().path)
    }

    @Test fun `offline mirror saves only referenced supported images once`() {
        val entries = listOf(
            WikiEntry("page", "one", "Notes/One.md", "One", "md"),
            WikiEntry("page", "two", "Notes/Two.md", "Two", "md"),
            WikiEntry("asset", "map", "Notes/map.png", "map", "png", revision = "r1"),
            WikiEntry("asset", "unused", "unused.jpg", "unused", "jpg", revision = "r1"),
        )
        val pages = listOf(
            WikiOfflinePage("Notes/One.md", WikiPage("one", "One", "![[map.png]]", "1", false), 1),
            WikiOfflinePage("Notes/Two.md", WikiPage("two", "Two", "![Map](map.png)", "1", false), 2),
        )
        assertEquals(listOf("map"), wikiOfflineAssetTargets(entries, pages).map(WikiEntry::id))
    }

    @Test fun `offline asset identity changes with its server revision`() {
        val first = WikiEntry("asset", "map", "map.png", "map", "png", revision = "r1")
        assertEquals(wikiOfflineAssetFileName(first), wikiOfflineAssetFileName(first.copy()))
        assertTrue(wikiOfflineAssetFileName(first) != wikiOfflineAssetFileName(first.copy(revision = "r2")))
    }

    @Test fun `offline status reports partial image coverage honestly`() {
        val result = WikiOfflineMirrorResult(4, 4, false, 2, 3, true, emptyMap())
        assertEquals("4 pages are available offline. 2 of 3 embedded images were saved.", buildOfflineWikiStatus(result))
    }
}
