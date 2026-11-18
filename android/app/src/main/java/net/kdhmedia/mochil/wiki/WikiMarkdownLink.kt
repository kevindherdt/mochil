package net.kdhmedia.mochil.wiki

import java.net.URLDecoder

/** Relative note/PDF links stay inside the indexed vault; external URLs retain browser handling. */
internal fun wikiMarkdownLink(label: String, destination: String): WikiInlineSegment? {
    if (destination.startsWith("https://", true) || destination.startsWith("http://", true)) {
        return WikiInlineSegment(label, destination, WikiInlineStyle.Link)
    }
    val target = runCatching { URLDecoder.decode(destination.replace("+", "%2B"), "UTF-8") }.getOrNull() ?: return null
    if (target.startsWith('/') || target.contains(':') || target.contains('\\') || target.any { it.code < 32 || it.code == 127 }) return null
    val path = target.substringBefore('#')
    if (!target.startsWith('#') && !path.endsWith(".md", true) && !path.endsWith(".pdf", true)) return null
    return WikiInlineSegment(label, target, WikiInlineStyle.WikiLink)
}
