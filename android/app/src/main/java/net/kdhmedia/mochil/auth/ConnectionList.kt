package net.kdhmedia.mochil.auth

/** The list with [connection] added or replaced; a new entry past [max] is refused instead of being dropped when saving. */
internal fun <T> connectionsWith(existing: List<T>, connection: T, max: Int, idOf: (T) -> String): List<T> {
    val others = existing.filterNot { idOf(it) == idOf(connection) }
    require(others.size < max) { "You can keep up to $max connections. Remove one first." }
    return others + connection
}

/** A saved change that did not reach storage must surface, not look like success. */
internal fun requireSaved(saved: Boolean) = check(saved) { "Could not save this change on the device." }
