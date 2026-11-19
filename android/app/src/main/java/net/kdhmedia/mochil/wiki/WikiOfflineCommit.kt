package net.kdhmedia.mochil.wiki

import java.io.IOException

/** Never evict usable caches until both the replacement and its index are durable. */
internal fun commitWikiOfflineCache(
    writePayload: () -> Boolean,
    writeIndex: () -> Boolean,
    evict: () -> Unit,
) {
    if (!writePayload()) throw IOException("Could not save the offline Wiki copy")
    if (!writeIndex()) throw IOException("Could not save the offline Wiki index")
    evict()
}
