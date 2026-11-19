package net.kdhmedia.mochil.catalogue

/** Keeps shelf and opened-group requests from publishing into a newer context. */
internal class LibraryRequestGeneration {
    private var shelf = 0L
    private var group = 0L

    fun advanceShelf(): Long = next(shelf).also { shelf = it }
    fun captureShelf(): Long = shelf
    fun isCurrentShelf(candidate: Long): Boolean = candidate == shelf

    fun advanceGroup(): Long = next(group).also { group = it }
    fun captureGroup(): Long = group
    fun isCurrentGroup(candidate: Long): Boolean = candidate == group

    private fun next(value: Long): Long = if (value == Long.MAX_VALUE) 1L else value + 1L
}
