package net.kdhmedia.mochil.wiki

import org.junit.Assert.assertEquals
import org.junit.Test

class WikiInlineTest {
    @Test fun `single matcher preserves previous formatting precedence and text`() {
        val fragments = listOf(
            "", "plain ", "[[folder/Page|Label]]", "[[folder/Page]]",
            "[link](HTTPS://example.test/path)", "**bold**", "*italic*", "~~strike~~",
            "` code [[link]] **bold** `", "**[[inside]]**", "[[Page|**label**]]",
            "***nested***", "[broken](", "\n", "[[missing", "雪😀", "` spaced `",
        )
        for (first in fragments) for (second in fragments) for (third in fragments.take(5)) {
            val text = first + second + third
            assertEquals(text, parseWikiInlineReference(text), parseWikiInline(text))
        }
    }

    @Test fun `many inline spans retain their order without combining adjacent styles`() {
        val text = (0 until 5_000).joinToString(" ") { "**item-$it**" }
        val result = parseWikiInline(text)
        assertEquals(9_999, result.size)
        assertEquals(WikiInlineSegment("item-0", style = WikiInlineStyle.Bold), result.first())
        assertEquals(WikiInlineSegment("item-4999", style = WikiInlineStyle.Bold), result.last())
    }
}

// Frozen prior implementation provides a differential compatibility oracle.
private val wikiLinkPattern = Regex("""\[\[([^\]\n|]+)(?:\|([^\]\n]+))?]]""")
private val markdownLinkPattern = Regex("""\[([^\]\n]+)]\((https?://[^\s)\n]+)\)""", RegexOption.IGNORE_CASE)
private val boldPattern = Regex("""\*\*([^*\n]+)\*\*""")
private val italicPattern = Regex("""(?<!\*)\*([^*\n]+)\*(?!\*)""")
private val strikethroughPattern = Regex("""~~([^~\n]+)~~""")
private val inlineCodePattern = Regex("""`([^`\n]+)`""")

private fun parseWikiInlineReference(text: String): List<WikiInlineSegment> {
    val result = mutableListOf<WikiInlineSegment>()
    var cursor = 0
    while (cursor < text.length) {
        val candidates = listOfNotNull(
            wikiLinkPattern.find(text, cursor)?.let { Triple(it, WikiInlineStyle.WikiLink, 0) },
            markdownLinkPattern.find(text, cursor)?.let { Triple(it, WikiInlineStyle.Link, 1) },
            boldPattern.find(text, cursor)?.let { Triple(it, WikiInlineStyle.Bold, 2) },
            strikethroughPattern.find(text, cursor)?.let { Triple(it, WikiInlineStyle.Strikethrough, 3) },
            inlineCodePattern.find(text, cursor)?.let { Triple(it, WikiInlineStyle.Code, 4) },
            italicPattern.find(text, cursor)?.let { Triple(it, WikiInlineStyle.Italic, 5) },
        )
        val (match, style) = candidates.minWithOrNull(compareBy<Triple<MatchResult, WikiInlineStyle, Int>> { it.first.range.first }.thenBy { it.third })
            ?.let { it.first to it.second } ?: break
        if (match.range.first > cursor) result += WikiInlineSegment(text.substring(cursor, match.range.first))
        val first = match.groupValues[1].trim()
        val second = match.groupValues.getOrElse(2) { "" }.trim()
        result += when (style) {
            WikiInlineStyle.WikiLink -> WikiInlineSegment(second.ifBlank { first.substringAfterLast('/') }, first, style)
            WikiInlineStyle.Link -> WikiInlineSegment(first, second, style)
            else -> WikiInlineSegment(first, style = style)
        }
        cursor = match.range.last + 1
    }
    if (cursor < text.length) result += WikiInlineSegment(text.substring(cursor))
    return result.ifEmpty { listOf(WikiInlineSegment(text)) }
}
