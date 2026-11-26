package net.kdhmedia.mochil.catalogue

/**
 * The reading-status chips above a shelf: All, In Progress, Unread (web/src/features/library/screens/LibraryScreen.tsx).
 * They filter the plain list of books on the server through `?status=`, so totals and paging describe the filtered shelf.
 * Finished books are the web's fourth value; the chips do not offer it, so neither does the app.
 */
enum class BookStatusFilter(val wire: String, val label: String) {
    All("", "All"),
    InProgress("in_progress", "In Progress"),
    Unread("unread", "Unread");

    /** What an empty shelf says under this filter, in the web's words. [noun] is what the shelf holds ("books", "audiobooks"). */
    fun emptyMessage(noun: String): String = when (this) {
        All -> "This shelf has no $noun."
        InProgress -> "No $noun in progress."
        Unread -> "Every item here has been opened."
    }

    companion object {
        /** Only the values the server accepts are known; anything else is All, so a stray string is never sent. */
        fun fromWire(wire: String): BookStatusFilter = entries.firstOrNull { it.wire.isNotEmpty() && it.wire == wire } ?: All
    }
}
