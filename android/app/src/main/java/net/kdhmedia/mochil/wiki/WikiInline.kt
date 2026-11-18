package net.kdhmedia.mochil.wiki

// Alternatives retain the previous tie precedence: wiki link, URL, bold, strike, code, italic.
private val inlinePattern = Regex(
    listOf(
        """\[\[([^\]\n|]+)(?:\|([^\]\n]+))?]]""",
        """\[([^\]\n]+)]\((https?://[^\s)\n]+|[^\s()\[\]\n]+)\)""",
        """\*\*([^*\n]+)\*\*""",
        """~~([^~\n]+)~~""",
        """`([^`\n]+)`""",
        """(?<!\*)\*([^*\n]+)\*(?!\*)""",
    ).joinToString("|"),
    RegexOption.IGNORE_CASE,
)

internal fun parseWikiInline(text: String): List<WikiInlineSegment> {
    val result = mutableListOf<WikiInlineSegment>()
    var cursor = 0
    for (match in inlinePattern.findAll(text)) {
        if (match.range.first > cursor) result += WikiInlineSegment(text.substring(cursor, match.range.first))
        val groups = match.groups
        result += when {
            groups[1] != null -> {
                val target = groups[1]!!.value.trim()
                val label = groups[2]?.value.orEmpty().trim().ifBlank { target.substringAfterLast('/') }
                WikiInlineSegment(label, target, WikiInlineStyle.WikiLink)
            }
            groups[3] != null -> wikiMarkdownLink(groups[3]!!.value.trim(), groups[4]!!.value.trim())
                ?: WikiInlineSegment(match.value, style = WikiInlineStyle.Plain)
            groups[5] != null -> WikiInlineSegment(groups[5]!!.value.trim(), style = WikiInlineStyle.Bold)
            groups[6] != null -> WikiInlineSegment(groups[6]!!.value.trim(), style = WikiInlineStyle.Strikethrough)
            groups[7] != null -> WikiInlineSegment(groups[7]!!.value.trim(), style = WikiInlineStyle.Code)
            else -> WikiInlineSegment(groups[8]!!.value.trim(), style = WikiInlineStyle.Italic)
        }
        cursor = match.range.last + 1
    }
    if (cursor < text.length) result += WikiInlineSegment(text.substring(cursor))
    return result.ifEmpty { listOf(WikiInlineSegment(text)) }
}
