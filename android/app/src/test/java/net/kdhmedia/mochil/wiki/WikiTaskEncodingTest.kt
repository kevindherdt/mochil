package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiTaskEncodingTest {
    @Test fun `task toggle skips fenced examples and preserves BOM and CRLF`() {
        val source = "\uFEFF---\r\ntitle: Tasks\r\n---\r\n```md\r\n- [ ] example\r\n```\r\n- [ ] real\r\n"
        val tasks = parseWikiMarkdown(source).filterIsInstance<WikiMarkdownBlock.ListItem>().filter { it.checked != null }
        assertEquals(1, tasks.size)
        assertEquals(source.replace("- [ ] real", "- [x] real"), toggleWikiTask(source, 0, true))
        assertEquals(source, toggleWikiTask(source, 1, true))
    }

    @Test fun `BOM does not hide first heading or first task`() {
        assertEquals(WikiMarkdownBlock.Heading(1, "Title"), parseWikiMarkdown("\uFEFF# Title\r\n").single())
        assertEquals("\uFEFF- [x] first\r\n", toggleWikiTask("\uFEFF- [ ] first\r\n", 0, true))
    }
}
