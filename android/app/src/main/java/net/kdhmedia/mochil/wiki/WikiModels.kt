package net.kdhmedia.mochil.wiki

data class WikiEntry(val kind: String, val id: String = "", val path: String, val title: String = "", val format: String = "", val matchSnippet: String = "", val revision: String = "", val modified: String = "") {
    val isFolder get() = kind == "folder"
    val depth get() = path.count { it == '/' }
    val label get() = if (isFolder) path.substringAfterLast('/') else title.ifBlank { path.substringAfterLast('/').substringBeforeLast('.') }
}

data class WikiPage(val id: String, val title: String, val content: String, val revision: String, val writable: Boolean)

data class WikiState(
    val libraryId: String = "",
    val entries: List<WikiEntry> = emptyList(),
    val expanded: Set<String> = emptySet(),
    val page: WikiPage? = null,
    val backlinks: List<WikiEntry> = emptyList(),
    val searchQuery: String = "",
    val searchResults: List<WikiEntry> = emptyList(),
    val searching: Boolean = false,
    val draftRecovered: Boolean = false,
    val navigationHeading: String = "",
    val missingLink: WikiMissingLink? = null,
    val selectedPath: String = "",
    val draft: String = "",
    val loading: Boolean = false,
    val saving: Boolean = false,
    val editing: Boolean = false,
    val offlineCopy: Boolean = false,
    val offlineSaving: Boolean = false,
    val offlineSaved: Int = 0,
    val offlineTotal: Int = 0,
    val offlineStatus: String? = null,
    val offlineAssetUrls: Map<String, String> = emptyMap(),
    val attachmentStatus: String? = null,
    val error: String? = null,
)

internal data class WikiLinkDestination(val page: String, val heading: String = "")
data class WikiMissingLink(val label: String, val path: String)

internal fun parseWikiLinkDestination(target: String): WikiLinkDestination {
    val normalized = target.trim().replace('\\', '/').substringBefore('|').trim()
    val page = normalized.substringBefore('#').substringBefore('^').trim().let {
        if (it.endsWith(".md", ignoreCase = true)) it.dropLast(3) else it
    }
    val heading = normalized.substringAfter('#', "").substringBefore('^').trim()
    return WikiLinkDestination(page, heading)
}

internal fun WikiState.withBacklinksIfCurrent(libraryId: String, pageId: String, values: List<WikiEntry>): WikiState =
    if (this.libraryId == libraryId && page?.id == pageId) copy(backlinks = values) else this

internal fun WikiState.withOfflinePageIfCurrent(libraryId: String, path: String, cached: WikiOfflinePage?): WikiState =
    if (cached != null && this.libraryId == libraryId && selectedPath == path) copy(
        page = cached.page,
        draft = cached.page.content,
        loading = true,
        offlineCopy = true,
    ) else this

/** Keep filename/path filtering usable when full-text search is unavailable. */
internal fun WikiState.withWikiSearchFailure(libraryId: String, query: String, message: String): WikiState =
    if (this.libraryId == libraryId && searchQuery == query) copy(
        searchQuery = "",
        searchResults = emptyList(),
        searching = false,
        error = message.takeUnless { offlineCopy },
    ) else this

internal fun normalizeWikiPagePath(value: String): String = value.trim().replace('\\', '/').trim('/').let {
    if (it.endsWith(".md", ignoreCase = true)) it else "$it.md"
}

/** Attachments remain indexed for safe embeds but are not presented as pages. */
internal fun wikiPageTreeEntries(entries: List<WikiEntry>): List<WikiEntry> = entries.filter {
    it.isFolder || it.format.equals("md", ignoreCase = true)
}

internal fun normalizeWikiAssetPath(value: String): String? {
    val parts = mutableListOf<String>()
    for (part in value.split('/')) when (part) {
        "", "." -> Unit
        ".." -> if (parts.isEmpty()) return null else parts.removeAt(parts.lastIndex)
        else -> parts += part
    }
    return parts.joinToString("/").takeIf(String::isNotBlank)
}

internal val WIKI_IMAGE_FORMATS = setOf("png", "jpg", "jpeg", "webp", "gif", "avif")

/**
 * Obsidian-style simple links create beside the current page. Links containing
 * a folder use that vault-relative path explicitly. Traversal is rejected.
 */
internal fun missingWikiLinkTarget(currentPagePath: String, target: String): WikiMissingLink? {
    val page = parseWikiLinkDestination(target).page
    if (page.isBlank() || page.startsWith('/') || page.contains(':') ||
        page.any { it.code < 32 || it.code == 127 } ||
        page.split('/').any { it.isBlank() || it == "." || it == ".." }) return null
    val parent = currentPagePath.substringBeforeLast('/', "")
    val relativePath = if ('/' in page || parent.isBlank()) page else "$parent/$page"
    // Parsing removed exactly one extension. Restore it even if the literal
    // filename itself ends in .md (for example Report.md.md).
    return WikiMissingLink(page.substringAfterLast('/'), "$relativePath.md")
}
