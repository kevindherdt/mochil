package net.kdhmedia.mochil.ui

private val releaseTags = Regex(
    """[\s._-]*[\[(]?\b(?:blu-?ray|bdrip|brrip|web[-\s.]?dl|webrip|hdtv|dvdrip|remux)\b[-\s.]*(?:\d{3,4}p)?[\])]?""",
    RegexOption.IGNORE_CASE,
)
private val specTags = Regex("""[\s._-]*\b(?:2160p|1080p|720p|480p|4k|hdr10?|dv|x26[45]|hevc|aac|ddp?5\.1)\b""", RegexOption.IGNORE_CASE)
private val bracketTags = Regex("""\s*\[[^\]]*]""")

/** A title as a person would say it: file-name leftovers such as "Bluray-1080p" or "[tvdbid=1]" are dropped on TV. */
internal fun tvCleanTitle(raw: String): String {
    val cleaned = raw.replace(bracketTags, "").replace(releaseTags, "").replace(specTags, "").trim().trimEnd('-', '.', '_', ' ')
    return cleaned.ifBlank { raw.trim() }
}

internal fun tvCount(count: Int, singular: String, plural: String = singular + "s"): String = "$count ${if (count == 1) singular else plural}"
