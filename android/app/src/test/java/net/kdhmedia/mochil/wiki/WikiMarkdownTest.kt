package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiMarkdownTest {
    @Test fun `parses bounded obsidian markdown tables and escaped pipes`() {
        val source = """| Name | Status |
            | --- | :---: |
            | [[Plan]] | Ready |
            | A \| B | Done |
        """.trimIndent()
        assertEquals(
            WikiMarkdownBlock.Table(
                listOf("Name", "Status"),
                listOf(listOf("[[Plan]]", "Ready"), listOf("A | B", "Done")),
            ),
            parseWikiMarkdown(source).single(),
        )
    }

    @Test fun `ordinary pipe text is not interpreted as a table`() {
        assertEquals(WikiMarkdownBlock.Paragraph("A | B is ordinary text"), parseWikiMarkdown("A | B is ordinary text").single())
    }

    @Test fun `table rendering is bounded without leaking surplus rows into paragraphs`() {
        val source = buildString {
            appendLine("A | B")
            appendLine("--- | ---")
            repeat(140) { appendLine("$it | value") }
        }
        val table = parseWikiMarkdown(source).single() as WikiMarkdownBlock.Table
        assertEquals(100, table.rows.size)
    }

    @Test fun `content after a table remains a separate block`() {
        val blocks = parseWikiMarkdown("A | B\n--- | ---\none | two\n\nAfter")
        assertTrue(blocks[0] is WikiMarkdownBlock.Table)
        assertEquals(WikiMarkdownBlock.Paragraph("After"), blocks[1])
    }

    @Test fun `parser preserves headings lists paragraphs and fenced code`() {
        val blocks = parseWikiMarkdown("# Title\n\nHello world\n\n- one\n\n```sh\necho hi\n```")
        assertEquals(4, blocks.size)
        assertEquals(WikiMarkdownBlock.Heading(1, "Title"), blocks[0])
        assertTrue(blocks[2] is WikiMarkdownBlock.ListItem)
        assertEquals(WikiMarkdownBlock.Code("sh", "echo hi"), blocks[3])
    }

    @Test fun `inline parser preserves Obsidian targets and aliases`() {
        assertEquals(
            listOf(WikiInlineSegment("See "), WikiInlineSegment("Plan", "Projects/Plan"), WikiInlineSegment(" and "), WikiInlineSegment("home", "Home")),
            parseWikiInline("See [[Projects/Plan]] and [[Home|home]]"),
        )
    }

    @Test fun `inline parser honors every format produced by the editor`() {
        assertEquals(
            listOf(
                WikiInlineSegment("Bold", style = WikiInlineStyle.Bold),
                WikiInlineSegment(" and "),
                WikiInlineSegment("italic", style = WikiInlineStyle.Italic),
                WikiInlineSegment(" with "),
                WikiInlineSegment("old", style = WikiInlineStyle.Strikethrough),
                WikiInlineSegment(", "),
                WikiInlineSegment("code", style = WikiInlineStyle.Code),
                WikiInlineSegment(" and "),
                WikiInlineSegment("site", "https://example.com", WikiInlineStyle.Link),
            ),
            parseWikiInline("**Bold** and *italic* with ~~old~~, `code` and [site](https://example.com)"),
        )
    }

    @Test fun `ordinary links accept only explicit web schemes`() {
        assertEquals(
            listOf(WikiInlineSegment("[unsafe](javascript:alert)")),
            parseWikiInline("[unsafe](javascript:alert)"),
        )
    }

    @Test fun `parser preserves common Obsidian document blocks`() {
        val blocks = parseWikiMarkdown(
            """> A useful note
            |
            |- [ ] open task
            |  - [x] nested completed task
            |1. first step
            |
            |---""".trimMargin(),
        )
        assertEquals(WikiMarkdownBlock.Quote("A useful note"), blocks[0])
        assertEquals(WikiMarkdownBlock.ListItem("open task", checked = false, taskOrdinal = 0), blocks[1])
        assertEquals(WikiMarkdownBlock.ListItem("nested completed task", checked = true, depth = 1, taskOrdinal = 1), blocks[2])
        assertEquals(WikiMarkdownBlock.ListItem("first step", marker = "1."), blocks[3])
        assertEquals(WikiMarkdownBlock.Rule, blocks[4])
    }

    @Test fun `parser preserves Obsidian and Markdown image embeds as safe targets`() {
        val blocks = parseWikiMarkdown("![[Assets/cover.png|Cover art]]\n\n![Diagram](../Images/map.webp)")
        assertEquals(WikiMarkdownBlock.Image("Assets/cover.png", "Cover art"), blocks[0])
        assertEquals(WikiMarkdownBlock.Image("../Images/map.webp", "Diagram"), blocks[1])
    }

    @Test fun `Obsidian frontmatter becomes properties instead of document prose`() {
        val blocks = parseWikiMarkdown("---\ntitle: Plan\ntags: [work, home]\n---\n# Body")
        assertEquals(
            WikiMarkdownBlock.Properties(listOf(WikiProperty("title", "Plan"), WikiProperty("tags", "[work, home]"))),
            blocks[0],
        )
        assertEquals(WikiMarkdownBlock.Heading(1, "Body"), blocks[1])
    }

    @Test fun `an unfinished frontmatter fence remains ordinary Markdown`() {
        assertEquals(WikiMarkdownBlock.Rule, parseWikiMarkdown("---\ntitle: Plan")[0])
    }

    @Test fun `task toggle changes only the requested marker and preserves line endings`() {
        val source = "---\r\ntitle: Plan\r\n---\r\n- [ ] first\r\n  - [X] second\r\n"
        assertEquals(
            "---\r\ntitle: Plan\r\n---\r\n- [ ] first\r\n  - [ ] second\r\n",
            toggleWikiTask(source, 1, false),
        )
        assertEquals(source, toggleWikiTask(source, 8, true))
    }

    @Test fun `task-looking frontmatter values do not affect rendered task identity`() {
        val source = "---\nexample: - [ ] metadata\n---\n- [ ] real task\n"
        assertEquals("---\nexample: - [ ] metadata\n---\n- [x] real task\n", toggleWikiTask(source, 0, true))
    }

    @Test fun `outline retains heading hierarchy and rendered block positions`() {
        assertEquals(
            listOf(WikiOutlineEntry(1, "Start", 1), WikiOutlineEntry(3, "Details", 3)),
            wikiOutline("---\ntitle: Plan\n---\n# Start\n\nText\n\n### Details"),
        )
    }

    @Test fun `empty frontmatter is omitted without hiding the body or shifting tasks`() {
        for (newline in listOf("\n", "\r\n")) {
            for (prefix in listOf("", "\uFEFF")) {
                val source = "$prefix---$newline---$newline- [ ] body"
                val blocks = parseWikiMarkdown(source)
                assertEquals(1, blocks.size)
                val task = blocks.single() as WikiMarkdownBlock.ListItem
                assertEquals("body", task.text)
                assertEquals(0, task.taskOrdinal)
                assertEquals(source.replace("- [ ]", "- [x]"), toggleWikiTask(source, 0, true))
                assertEquals(emptyList<WikiProperty>() to "", extractWikiFrontmatter("$prefix---$newline---"))
            }
        }
    }

    @Test fun `frontmatter delimiter prefix cannot disagree with task numbering`() {
        for (newline in listOf("\n", "\r\n")) {
            val source = listOf("---", "title: Plan", "---not-a-delimiter", "- [ ] metadata", "---", "- [ ] body").joinToString(newline)
            val tasks = parseWikiMarkdown(source).filterIsInstance<WikiMarkdownBlock.ListItem>()
            assertEquals(1, tasks.size)
            assertEquals("body", tasks.single().text)
            assertEquals(0, tasks.single().taskOrdinal)
            assertEquals(source.replace("- [ ] body", "- [x] body"), toggleWikiTask(source, 0, true))
        }
    }

    @Test fun `parsed document shares one block list with its outline`() {
        val document = parseWikiMarkdownDocument("# Start\n\nText\n\n## Details")
        assertEquals(3, document.blocks.size)
        assertEquals(
            listOf(WikiOutlineEntry(1, "Start", 0), WikiOutlineEntry(2, "Details", 2)),
            document.outline,
        )
    }

    @Test fun `streaming parser observes cancellation checkpoints`() {
        var checks = 0
        val stopped = runCatching {
            parseWikiMarkdown((1..1_000).joinToString("\n") { "line $it" }) {
                checks++
                if (checks == 8) error("cancelled")
            }
        }.exceptionOrNull()
        assertEquals("cancelled", stopped?.message)
        assertEquals(8, checks)
    }
}
