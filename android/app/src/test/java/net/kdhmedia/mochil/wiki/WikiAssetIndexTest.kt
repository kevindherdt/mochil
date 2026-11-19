package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiAssetIndexTest {
    private val entries = listOf(
        WikiEntry("page", "other", "Other/cover.png", "cover", "png"),
        WikiEntry("page", "local", "Notes/Assets/cover.png", "cover", "png"),
    )

    @Test fun `explicit missing paths never fall back to an unrelated filename`() {
        val index = WikiAssetIndex(entries)
        for (target in listOf("Missing/cover.png", "./cover.png", "../Missing/cover.png", "Missing\\cover.png", "Missing/cover.png|300")) {
            assertNull(target, index.resolve("Notes/Page.md", target))
        }
    }

    @Test fun `explicit relative and vault paths still resolve and shorthand remains supported`() {
        val index = WikiAssetIndex(entries)
        assertEquals("local", index.resolve("Notes/Page.md", "Assets/cover.png")?.id)
        assertEquals("local", index.resolve("Elsewhere/Page.md", "Notes/Assets/cover.png")?.id)
        assertEquals("other", index.resolve("Notes/Page.md", "../Other/cover.png")?.id)
        assertNull(index.resolve("Notes/Page.md", "cover.png"))
    }

    @Test fun `local image wins over root regardless of index order`() {
        val images = listOf(
            WikiEntry("page", "root", "cover.png", "cover", "png"),
            WikiEntry("page", "local", "Notes/cover.png", "cover", "png"),
        )
        for (ordered in listOf(images, images.reversed())) {
            val index = WikiAssetIndex(ordered)
            assertEquals("local", index.resolve("Notes/Page.md", "cover.png")?.id)
            assertEquals("local", index.resolve("Notes/Page.md", "./cover.png")?.id)
            assertEquals("root", index.resolve("Notes/Page.md", "../cover.png")?.id)
            assertNull(index.resolve("Missing/Page.md", "./cover.png"))
            assertNull(index.resolve("Notes/Page.md", "../../cover.png"))
        }
    }

    @Test fun `ambiguous folded paths and unsafe targets remain unresolved`() {
        val index = WikiAssetIndex(listOf(
            WikiEntry("page", "upper", "Notes/COVER.png", "cover", "png"),
            WikiEntry("page", "lower", "Notes/cover.png", "cover", "png"),
        ))
        assertEquals("lower", index.resolve("Notes/Page.md", "cover.png")?.id)
        assertEquals("upper", index.resolve("Notes/Page.md", "COVER.png")?.id)
        for (target in listOf("Cover.png", "data:cover.png", "javascript:cover.png", "bad\u0000.png", "bad\u007f.png")) {
            assertNull(target, index.resolve("Notes/Page.md", target))
        }
    }
}
