package net.kdhmedia.mochil.audio

/**
 * Selects sidecar-indexed downloads that can safely rebuild app-private state.
 * Existing app records always win, and a damaged index cannot add the same
 * media identity more than once.
 */
internal fun <T> recoverableExternalDownloads(
    indexed: List<T>,
    currentKeys: Set<String>,
    keyOf: (T) -> String,
    isReadable: (T) -> Boolean,
): List<T> = indexed.asSequence()
    .filter(isReadable)
    .distinctBy(keyOf)
    .filterNot { keyOf(it) in currentKeys }
    .toList()
