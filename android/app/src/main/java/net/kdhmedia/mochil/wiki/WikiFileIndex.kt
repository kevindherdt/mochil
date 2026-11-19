package net.kdhmedia.mochil.wiki

private val wikiAssetScheme = Regex("^[a-zA-Z][a-zA-Z0-9+.-]*:")

/** Resolves only indexed files in explicitly allowed formats; never constructs a remote URL. */
internal class WikiFileIndex(entries: List<WikiEntry>, formats: Set<String>) {
    private val byExactPath: Map<String, WikiEntry?>
    private val byPath: Map<String, WikiEntry?>
    private val byName: Map<String, WikiEntry?>

    init {
        val exactPaths = LinkedHashMap<String, WikiEntry?>()
        val paths = LinkedHashMap<String, WikiEntry?>()
        val names = LinkedHashMap<String, WikiEntry?>()
        fun add(index: MutableMap<String, WikiEntry?>, key: String, entry: WikiEntry) {
            if (!index.containsKey(key)) index[key] = entry
            else if (index[key]?.id != entry.id) index[key] = null
        }
        entries.forEach { entry ->
            if (!entry.isFolder && entry.id.isNotBlank() && entry.format.lowercase() in formats) {
                add(exactPaths, entry.path, entry)
                add(paths, entry.path.lowercase(), entry)
                add(names, entry.path.substringAfterLast('/').lowercase(), entry)
            }
        }
        byExactPath = exactPaths
        byPath = paths
        byName = names
    }

    fun resolve(pagePath: String, target: String): WikiEntry? {
        val cleanTarget = target.substringBefore('#').substringBefore('|').trim().replace('\\', '/')
        if (cleanTarget.isBlank() || wikiAssetScheme.containsMatchIn(cleanTarget) ||
            cleanTarget.startsWith('/') || cleanTarget.any { it.code < 32 || it.code == 127 }) return null
        val parent = pagePath.substringBeforeLast('/', "")
        val relative = normalizeWikiAssetPath(if (parent.isBlank()) cleanTarget else "$parent/$cleanTarget") ?: return null
        val candidates = buildList {
            add(relative)
            if (!cleanTarget.startsWith("./") && !cleanTarget.startsWith("../")) normalizeWikiAssetPath(cleanTarget)?.let(::add)
        }
        for (candidate in candidates) {
            if (byExactPath.containsKey(candidate)) return byExactPath[candidate]
            val key = candidate.lowercase()
            if (byPath.containsKey(key)) return byPath[key]
        }
        // A named directory is intent, not a hint: never substitute an unrelated
        // same-named attachment when an explicit path is broken.
        if ('/' in cleanTarget) return null
        return byName[cleanTarget.lowercase()]
    }
}
