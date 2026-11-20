package net.kdhmedia.mochil.wiki

/** Folder-first natural depth-first order, without consuming the thread call stack. */
internal fun orderedWikiEntries(entries: List<WikiEntry>): List<WikiEntry> {
    val children = entries.filter { it.path.isNotBlank() }.groupBy { it.path.substringBeforeLast('/', "") }
    val result = ArrayList<WikiEntry>(entries.size)
    val pending = ArrayDeque<Iterator<WikiEntry>>()
    pending.addLast(orderedSiblings(children[""].orEmpty()).iterator())
    while (pending.isNotEmpty()) {
        val siblings = pending.last()
        if (!siblings.hasNext()) {
            pending.removeLast()
            continue
        }
        val entry = siblings.next()
        result += entry
        if (entry.isFolder) children[entry.path]?.let {
            pending.addLast(orderedSiblings(it).iterator())
        }
    }
    return result
}

private fun orderedSiblings(entries: List<WikiEntry>): List<WikiEntry> {
    if (entries.size < 2) return entries
    return entries.map { it to wikiNaturalNameParts(it.label) }.sortedWith { a, b ->
        val (left, leftParts) = a
        val (right, rightParts) = b
        when {
            left.isFolder != right.isFolder -> if (left.isFolder) -1 else 1
            else -> compareWikiNaturalParts(leftParts, rightParts).takeIf { it != 0 }
                ?: left.path.compareTo(right.path, ignoreCase = true)
        }
    }.map { it.first }
}

internal fun naturalWikiNameCompare(left: String, right: String): Int =
    compareWikiNaturalParts(wikiNaturalNameParts(left), wikiNaturalNameParts(right))

private fun wikiNaturalNameParts(value: String) = WIKI_NATURAL_PART.findAll(value).map { it.value }.toList()

private fun compareWikiNaturalParts(leftParts: List<String>, rightParts: List<String>): Int {
    for (index in 0 until minOf(leftParts.size, rightParts.size)) {
        val a = leftParts[index]
        val b = rightParts[index]
        val comparison = if (a.firstOrNull()?.isDigit() == true && b.firstOrNull()?.isDigit() == true) {
            val normalizedA = a.trimStart('0').ifEmpty { "0" }
            val normalizedB = b.trimStart('0').ifEmpty { "0" }
            normalizedA.length.compareTo(normalizedB.length).takeIf { it != 0 }
                ?: normalizedA.compareTo(normalizedB).takeIf { it != 0 }
                ?: a.length.compareTo(b.length)
        } else a.compareTo(b, ignoreCase = true)
        if (comparison != 0) return comparison
    }
    return leftParts.size.compareTo(rightParts.size)
}

private val WIKI_NATURAL_PART = Regex("\\d+|\\D+")
