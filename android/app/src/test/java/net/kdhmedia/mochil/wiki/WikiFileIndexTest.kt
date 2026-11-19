package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiFileIndexTest {
    private val entries = listOf(
        WikiEntry("page", "root", "Manual.pdf", "Manual", "pdf"),
        WikiEntry("page", "local", "Notes/Manual.pdf", "Manual", "pdf"),
        WikiEntry("page", "upper", "Notes/MANUAL.pdf", "Manual", "pdf"),
        WikiEntry("page", "image", "Notes/cover.png", "cover", "png"),
        WikiEntry("page", "", "Notes/unidentified.pdf", "missing", "pdf"),
    )

    @Test fun `attachment policy and indexed identity are required`() {
        val index = WikiFileIndex(entries, setOf("pdf"))
        assertNull(index.resolve("Notes/Page.md", "cover.png"))
        assertNull(index.resolve("Notes/Page.md", "unidentified.pdf"))
        assertNull(WikiAssetIndex(entries).resolve("Notes/Page.md", "Manual.pdf"))
        assertEquals("image", WikiAssetIndex(entries).resolve("Notes/Page.md", "cover.png")?.id)
    }

    @Test fun `PDF references use local context exact case and bounded relative traversal`() {
        for (ordered in listOf(entries, entries.reversed())) {
            val index = WikiFileIndex(ordered, setOf("pdf"))
            assertEquals("local", index.resolve("Notes/Page.md", "Manual.pdf#page=3|Read manual")?.id)
            assertEquals("upper", index.resolve("Notes/Page.md", "MANUAL.pdf")?.id)
            assertEquals("root", index.resolve("Notes/Page.md", "../Manual.pdf")?.id)
            for (target in listOf("manual.pdf", "../../Manual.pdf", "/Manual.pdf", "https://host/Manual.pdf", "Missing/Manual.pdf")) {
                assertNull(target, index.resolve("Notes/Page.md", target))
            }
        }
    }
}
