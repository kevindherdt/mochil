package net.kdhmedia.mochil.wiki

private val wikiPageScheme = Regex("^[a-zA-Z][a-zA-Z0-9+.-]*:")

private data class WikiPageMatch(val found: Boolean, val entry: WikiEntry?)

private inline fun uniqueWikiPage(entries: List<WikiEntry>, matches: (WikiEntry) -> Boolean): WikiPageMatch {
    var selected: WikiEntry? = null
    for (entry in entries) {
        if (entry.isFolder || !entry.format.equals("md", ignoreCase = true) || !matches(entry)) continue
        if (selected != null && selected.id != entry.id) return WikiPageMatch(true, null)
        selected = entry
    }
    return WikiPageMatch(selected != null, selected)
}

private fun matchesWikiPagePath(entry: WikiEntry, target: String, ignoreCase: Boolean): Boolean {
    val length = entry.path.length - if (entry.path.endsWith(".md", ignoreCase = true)) 3 else 0
    return length == target.length && entry.path.regionMatches(0, target, 0, length, ignoreCase)
}

private fun matchesWikiPageName(entry: WikiEntry, target: String, ignoreCase: Boolean): Boolean {
    val start = entry.path.lastIndexOf('/') + 1
    val end = entry.path.length - if (entry.path.endsWith(".md", ignoreCase = true)) 3 else 0
    return end - start == target.length && entry.path.regionMatches(start, target, 0, target.length, ignoreCase)
}

/** Match local context first; ambiguous fallback labels never choose by scan order. */
internal fun findWikiLinkTarget(entries: List<WikiEntry>, target: String, currentPagePath: String = ""): WikiEntry? =
    findWikiLinkTarget(entries, parseWikiLinkDestination(target), currentPagePath)

/** A parsed destination must not have another literal filename suffix removed. */
internal fun findWikiLinkTarget(entries: List<WikiEntry>, destination: WikiLinkDestination, currentPagePath: String = ""): WikiEntry? {
    val normalized = destination.page
    if (normalized.isBlank() || normalized.startsWith('/') || wikiPageScheme.containsMatchIn(normalized) ||
        normalized.any { it.code < 32 || it.code == 127 }) return null
    val parent = currentPagePath.substringBeforeLast('/', "")
    val relative = normalizeWikiAssetPath(if (parent.isBlank()) normalized else "$parent/$normalized") ?: return null
    val candidates = buildList {
        add(relative)
        if (!normalized.startsWith("./") && !normalized.startsWith("../")) normalizeWikiAssetPath(normalized)?.let(::add)
    }.distinct()
    for (candidate in candidates) {
        val exact = uniqueWikiPage(entries) { matchesWikiPagePath(it, candidate, false) }
        if (exact.found) return exact.entry
        val folded = uniqueWikiPage(entries) { matchesWikiPagePath(it, candidate, true) }
        if (folded.found) return folded.entry
    }
    if ('/' in normalized) return null
    // A page's display title can differ from its filename. Short vault links
    // identify filenames first, including notes in another folder.
    val exactName = uniqueWikiPage(entries) { matchesWikiPageName(it, normalized, false) }
    if (exactName.found) return exactName.entry
    val foldedName = uniqueWikiPage(entries) { matchesWikiPageName(it, normalized, true) }
    if (foldedName.found) return foldedName.entry
    return uniqueWikiPage(entries) { it.label.equals(normalized, ignoreCase = true) }.entry
}
