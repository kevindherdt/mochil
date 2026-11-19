package net.kdhmedia.mochil.wiki

/**
 * Immutable, naturally ordered projection of one vault index.
 *
 * Building the hierarchy is the expensive part. Keeping it separate from the
 * expanded-folder projection lets a large vault open and close branches
 * without grouping and sorting every entry again.
 */
internal class WikiTreeIndex(entries: List<WikiEntry>) {
    private val ordered = orderedWikiEntries(wikiPageTreeEntries(entries))
    // Search has path order, not the natural folder order used for browsing.
    // Build it only when needed, computing each lowercase key once rather than
    // inside every comparator invocation on every keystroke.
    private val searchOrder by lazy {
        ordered.asSequence().filterNot(WikiEntry::isFolder)
            .map { it.path.lowercase() to it }
            .sortedBy { it.first }
            .map { it.second }
            .toList()
    }
    private val parents = run {
        // Ordering visits parents before children. Retain one integer per row,
        // not a copied list of every ancestor path at every nesting level.
        val folders = HashMap<String, Int>()
        IntArray(ordered.size) { index ->
            val entry = ordered[index]
            val parent = folders[entry.path.substringBeforeLast('/', "")] ?: -1
            if (entry.isFolder) folders[entry.path] = index
            parent
        }
    }

    fun visible(expanded: Set<String>): List<WikiEntry> {
        val childrenVisible = BooleanArray(ordered.size)
        return buildList {
            ordered.forEachIndexed { index, entry ->
                val parent = parents[index]
                val visible = parent < 0 || childrenVisible[parent]
                if (visible) add(entry)
                childrenVisible[index] = visible && entry.isFolder && entry.path in expanded
            }
        }
    }

    fun search(query: String, expanded: Set<String>): List<WikiEntry> {
        val needle = query.trim()
        if (needle.isBlank()) return visible(expanded)
        return searchOrder.asSequence()
            .filter { it.label.contains(needle, ignoreCase = true) || it.path.contains(needle, ignoreCase = true) }
            .toList()
    }
}
