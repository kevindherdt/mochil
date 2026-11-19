package net.kdhmedia.mochil.wiki

import java.io.IOException

/** Count the same JSON/UTF-8 representation as storage without rebuilding earlier batches. */
internal class WikiOfflinePageBudget(snapshot: WikiOfflineSnapshot) {
    private var bytes = encodeWikiOfflineSnapshot(snapshot.copy(pages = emptyList()))?.size
        ?: throw IOException("This vault exceeds the offline cache limits")
    private var count = 0

    fun accept(page: WikiOfflinePage): Boolean {
        if (count >= MAX_OFFLINE_PAGES) return false
        val encoded = encodeWikiOfflinePage(page) ?: return false
        val additional = encoded.toString().encodeToByteArray().size + if (count == 0) 0 else 1
        if (additional > MAX_OFFLINE_PAYLOAD_BYTES - bytes) return false
        bytes += additional
        count++
        return true
    }
}
