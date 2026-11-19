package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiMissingLinkSafetyTest {
    @Test fun `missing link creation preserves a literal markdown suffix`() {
        for (target in listOf("Report.md.md#Details|Read", "Archive/Report.md.md")) {
            val missing = requireNotNull(missingWikiLinkTarget("Projects/Start.md", target))
            val expected = if (target.startsWith("Archive/")) "Archive/Report.md.md" else "Projects/Report.md.md"
            assertEquals(expected, missing.path)
            val created = WikiEntry("page", "new", missing.path, "Report", "md")
            assertEquals(created, findWikiLinkTarget(listOf(created), target, "Projects/Start.md"))
        }
    }

    @Test fun `absolute paths schemes and control characters never propose a writable note`() {
        for (target in listOf(
            "/Private/Note", "//host/Note", "\\Private\\Note", "C:\\Notes\\Page",
            "https://host/Note", "file:Note", "data:Note", "javascript:Note",
            "Note\u0000title", "Note\u007ftitle", "Folder/", "Folder//Note", "../Note",
        )) assertNull(target, missingWikiLinkTarget("Projects/Roadmap.md", target))
    }

    @Test fun `ordinary Unicode notes aliases headings and explicit markdown names remain creatable`() {
        assertEquals(WikiMissingLink("Café 日本語", "Projects/Café 日本語.md"),
            missingWikiLinkTarget("Projects/Roadmap.md", "Café 日本語.md#Details|Read this"))
        assertEquals(WikiMissingLink("Page", "Reference/Page.md"),
            missingWikiLinkTarget("Projects/Roadmap.md", "Reference/Page"))
        assertEquals(WikiMissingLink("Version 1.2", "Projects/Version 1.2.md"),
            missingWikiLinkTarget("Projects/Roadmap.md", "Version 1.2"))
    }
}
