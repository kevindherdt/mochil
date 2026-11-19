package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiOfflineStoreTest {
    @Test fun `selected page survives count trimming even when downloaded first`() {
        val pages = List(MAX_OFFLINE_PAGES + 1) { index ->
            WikiOfflinePage("$index.md", WikiPage("$index", "Page", "text", "r", false), index.toLong())
        }
        val incoming = WikiOfflineSnapshot("vault", emptyList(), "0.md", pages, 200)
        val merged = mergeWikiOfflineSnapshots(null, incoming)
        assertEquals(MAX_OFFLINE_PAGES, merged.pages.size)
        assertEquals("0", merged.pages.first().page.id)
        assertEquals("text", merged.selectedPage()?.page?.content)
    }

    @Test fun `selected page survives byte trimming without choosing its stale copy`() {
        val selected = WikiOfflinePage("Selected.md", WikiPage("selected", "Selected", "s".repeat(1_600_000), "r2", false), 10)
        val other = WikiOfflinePage("Other.md", WikiPage("other", "Other", "o".repeat(1_600_000), "r", false), 20)
        val previous = WikiOfflineSnapshot("vault", emptyList(), "Other.md", listOf(selected.copy(page = selected.page.copy(content = "stale", revision = "r1"), savedAt = 1)), 1)
        val incoming = WikiOfflineSnapshot("vault", emptyList(), selected.path, listOf(selected, other), 30)
        val merged = mergeWikiOfflineSnapshots(previous, incoming)
        val restored = decodeWikiOfflineSnapshot(boundedWikiOfflinePayload(merged)!!)!!
        assertEquals(listOf("selected"), restored.pages.map { it.page.id })
        assertEquals("r2", restored.selectedPage()?.page?.revision)
        assertEquals(selected.page.content, restored.selectedPage()?.page?.content)
    }

    @Test fun `late older snapshot cannot replace newer page tree or selection`() {
        val newest = WikiOfflinePage("New.md", WikiPage("page", "New", "new text", "r2", true), 20)
        val stale = newest.copy(path = "Old.md", page = newest.page.copy(content = "old text", revision = "r1"), savedAt = 10)
        val current = WikiOfflineSnapshot("vault", listOf(WikiEntry("page", "page", "New.md")), "New.md", listOf(newest), 20)
        val incoming = WikiOfflineSnapshot("vault", emptyList(), "Old.md", listOf(stale), 10)
        val merged = mergeWikiOfflineSnapshots(current, incoming)
        assertEquals(current.entries, merged.entries)
        assertEquals(current.selectedPath, merged.selectedPath)
        assertEquals(listOf(newest), merged.pages)
        assertEquals(20L, merged.savedAt)
    }

    @Test fun `equal timestamps prefer incoming page as before`() {
        val old = WikiOfflinePage("Page.md", WikiPage("page", "Page", "old", "r1", true), 1)
        val incoming = old.copy(page = old.page.copy(content = "new", revision = "r2"))
        val snapshot = WikiOfflineSnapshot("vault", emptyList(), old.path, listOf(old), 1)
        assertEquals(listOf(incoming), mergeWikiOfflineSnapshots(snapshot, snapshot.copy(pages = listOf(incoming))).pages)
    }

    @Test fun `large recent page set keeps only fitting newest prefix`() {
        val content = "x".repeat(500_000)
        val pages = List(128) { index ->
            WikiOfflinePage("$index.md", WikiPage("$index", "Page", content, "r", true), (128-index).toLong())
        }
        val payload = requireNotNull(boundedWikiOfflinePayload(WikiOfflineSnapshot("vault", emptyList(), "0.md", pages, 128)))
        assertTrue(payload.size <= MAX_OFFLINE_PAYLOAD_BYTES)
        assertEquals(listOf("0", "1", "2", "3", "4"), decodeWikiOfflineSnapshot(payload)!!.pages.map { it.page.id })
    }

    @Test fun `byte budget includes unicode and JSON escaping`() {
        val content = "漢\n\"".repeat(300_000)
        val pages = List(2) { index -> WikiOfflinePage("$index.md", WikiPage("$index", "Page", content, "r", true), 1) }
        val snapshot = WikiOfflineSnapshot("vault", emptyList(), "0.md", pages, 1)
        assertNull(encodeWikiOfflineSnapshot(snapshot))
        val payload = requireNotNull(boundedWikiOfflinePayload(snapshot))
        assertTrue(payload.size <= MAX_OFFLINE_PAYLOAD_BYTES)
        val restored = decodeWikiOfflineSnapshot(payload)!!
        assertEquals(1, restored.pages.size)
        assertEquals(content, restored.pages.single().page.content)
    }

    @Test fun `offline snapshot round trips as read only`() {
        val snapshot = WikiOfflineSnapshot(
            "vault",
            listOf(WikiEntry("folder", path = "Notes"), WikiEntry("page", "page", "Notes/Plan.md", "Plan", "md")),
            "Notes/Plan.md",
            listOf(WikiOfflinePage("Notes/Plan.md", WikiPage("page", "Plan", "# Plan", "revision", true), 42)),
            42,
        )
        val decoded = decodeWikiOfflineSnapshot(encodeWikiOfflineSnapshot(snapshot)!!)!!
        assertEquals(snapshot.libraryId, decoded.libraryId)
        assertEquals(snapshot.entries, decoded.entries)
        assertEquals(snapshot.selectedPath, decoded.selectedPath)
        assertEquals(snapshot.pages.single().page.content, decoded.selectedPage()?.page?.content)
        assertFalse(decoded.selectedPage()!!.page.writable)
    }

    @Test fun `offline snapshot rejects oversized content and malformed data`() {
        val oversized = WikiOfflineSnapshot(
            "vault", emptyList(), "Page.md",
            listOf(WikiOfflinePage("Page.md", WikiPage("page", "Page", "x".repeat(2_000_001), "r", true), 0)), 0,
        )
        assertNull(encodeWikiOfflineSnapshot(oversized))
        assertNull(decodeWikiOfflineSnapshot("not-json".encodeToByteArray()))
    }

    @Test fun `recent pages merge by identity and evict the oldest`() {
        var snapshot: WikiOfflineSnapshot? = null
        repeat(130) { index ->
            val page = WikiPage("page-$index", "Page $index", "content $index", "r$index", true)
            snapshot = mergeWikiOfflineSnapshots(
                snapshot,
                WikiOfflineSnapshot("vault", emptyList(), "Page $index.md", listOf(WikiOfflinePage("Page $index.md", page, index.toLong())), index.toLong()),
            )
        }
        val result = requireNotNull(snapshot)
        assertEquals(128, result.pages.size)
        assertEquals("page-129", result.pages.first().page.id)
        assertEquals("page-2", result.pages.last().page.id)
        assertNull(result.page("page-1"))
    }

    @Test fun `newer copy replaces an existing cached page`() {
        val old = WikiOfflinePage("Page.md", WikiPage("page", "Page", "old", "r1", true), 1)
        val newer = WikiOfflinePage("Page.md", WikiPage("page", "Page", "new", "r2", true), 2)
        val merged = mergeWikiOfflineSnapshots(
            WikiOfflineSnapshot("vault", emptyList(), old.path, listOf(old), 1),
            WikiOfflineSnapshot("vault", emptyList(), newer.path, listOf(newer), 2),
        )
        assertEquals(1, merged.pages.size)
        assertEquals("new", merged.pages.single().page.content)
    }

    @Test fun `different vault snapshots never merge recent pages`() {
        val previous = WikiOfflineSnapshot(
            "private-vault", emptyList(), "Secret.md",
            listOf(WikiOfflinePage("Secret.md", WikiPage("secret", "Secret", "private", "r1", true), 1)), 1,
        )
        val incoming = WikiOfflineSnapshot(
            "shared-vault", emptyList(), "Welcome.md",
            listOf(WikiOfflinePage("Welcome.md", WikiPage("welcome", "Welcome", "shared", "r1", true), 2)), 2,
        )
        val merged = mergeWikiOfflineSnapshots(previous, incoming)
        assertEquals("shared-vault", merged.libraryId)
        assertEquals(listOf("welcome"), merged.pages.map { it.page.id })
    }

    @Test fun `vault storage identities are stable opaque and isolated`() {
        val first = wikiOfflineStorageIdentity("alice-device", "private-vault")
        assertEquals(first, wikiOfflineStorageIdentity("alice-device", "private-vault"))
        assertFalse(first.contains("private-vault"))
        assertFalse(first.contains("alice"))
        assertEquals(24, first.length)
        assertTrue(first.all { it in '0'..'9' || it in 'a'..'f' })
        assertFalse(first == wikiOfflineStorageIdentity("alice-device", "shared-vault"))
        assertFalse(first == wikiOfflineStorageIdentity("bob-device", "private-vault"))
        assertFalse(first == wikiLegacyStorageIdentity("private-vault"))
    }

    @Test fun `vault storage order stays bounded and evicts least recent slots`() {
        val current = listOf("01".repeat(12), "02".repeat(12), "03".repeat(12), "04".repeat(12))
        val newest = "05".repeat(12)
        val order = updateWikiOfflineStorageOrder(current, newest)
        assertEquals(listOf(newest) + current.take(3), order.kept)
        assertEquals(listOf(current.last()), order.evicted)

        val reused = updateWikiOfflineStorageOrder(order.kept, current[1])
        assertEquals(current[1], reused.kept.first())
        assertEquals(4, reused.kept.size)
        assertTrue(reused.evicted.isEmpty())
    }

    @Test fun `payload size evicts oldest pages before abandoning the cache`() {
        val pages = listOf(
            WikiOfflinePage("New.md", WikiPage("new", "New", "n".repeat(1_600_000), "r2", true), 2),
            WikiOfflinePage("Old.md", WikiPage("old", "Old", "o".repeat(1_600_000), "r1", true), 1),
        )
        val payload = boundedWikiOfflinePayload(WikiOfflineSnapshot("vault", emptyList(), "New.md", pages, 2))
        val decoded = decodeWikiOfflineSnapshot(requireNotNull(payload))!!
        assertEquals(listOf("new"), decoded.pages.map { it.page.id })
    }

    @Test fun `legacy single page cache migrates as read only`() {
        val legacy = """{"library_id":"vault","entries":[],"selected_path":"Page.md","page":{"id":"page","title":"Page","content":"legacy","revision":"r1"},"saved_at":42}"""
        val decoded = decodeWikiOfflineSnapshot(legacy.encodeToByteArray())!!
        assertEquals("legacy", decoded.selectedPage()?.page?.content)
        assertTrue(decoded.selectedPage()?.savedAt == 42L)
        assertFalse(decoded.selectedPage()!!.page.writable)
    }
}
