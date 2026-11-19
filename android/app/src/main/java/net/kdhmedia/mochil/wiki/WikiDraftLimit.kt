package net.kdhmedia.mochil.wiki

internal const val MAX_WIKI_DRAFT_CHARS = 2_000_000
internal const val WIKI_DRAFT_LIMIT_ERROR = "This edit is too large. Your previous text is unchanged; paste a smaller section or split it into another page."

internal fun acceptsWikiDraft(content: String): Boolean = content.length <= MAX_WIKI_DRAFT_CHARS
