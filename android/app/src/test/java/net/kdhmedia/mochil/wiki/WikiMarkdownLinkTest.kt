package net.kdhmedia.mochil.wiki

import org.junit.Assert.*
import org.junit.Test

class WikiMarkdownLinkTest {
    @Test fun `relative note PDF and heading Markdown links use vault navigation`() {
        for ((destination, expected) in listOf("Manual%20file.pdf" to "Manual file.pdf", "../Note.md#Details" to "../Note.md#Details", "#Details" to "#Details", "A+B.pdf" to "A+B.pdf")) {
            val link = parseWikiInline("[Open]($destination)").single()
            assertEquals(WikiInlineStyle.WikiLink, link.style)
            assertEquals(expected, link.target)
        }
        assertEquals(WikiInlineStyle.Link, parseWikiInline("[Site](https://example.org)").single().style)
    }

    @Test fun `unsafe and unsupported Markdown destinations remain inert text`() {
        for (destination in listOf("javascript:alert", "file:Manual.pdf", "/Manual.pdf", "%2FManual.pdf", "bad%00.pdf", "bad%GG.pdf", "image.png")) {
            val source = "[Open]($destination)"
            assertEquals(listOf(WikiInlineSegment(source, style = WikiInlineStyle.Plain)), parseWikiInline(source))
        }
    }
}
