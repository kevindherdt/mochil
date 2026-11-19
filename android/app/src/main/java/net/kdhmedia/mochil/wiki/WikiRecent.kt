package net.kdhmedia.mochil.wiki

/** The pages edited last, newest first: the Wiki's way back into what you were writing. */
internal fun recentWikiPages(entries: List<WikiEntry>, limit: Int = 5): List<WikiEntry> =
    entries.filter { !it.isFolder && it.modified.isNotBlank() && it.format.equals("md", ignoreCase = true) }
        .sortedByDescending { it.modified }
        .take(limit)
