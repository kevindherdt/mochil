package net.kdhmedia.mochil.wiki

/** Budgets the resulting cache, including retained files from earlier mirrors. */
internal fun boundedWikiAssetNames(candidates: List<Pair<String, Long>>): Set<String> {
    val keep = linkedSetOf<String>()
    var total = 0L
    for ((name, bytes) in candidates) {
        if (name in keep || bytes <= 0 || bytes > MAX_OFFLINE_ASSET_BYTES) continue
        if (keep.size >= MAX_OFFLINE_ASSETS) break
        if (total + bytes > MAX_OFFLINE_ASSET_TOTAL_BYTES) continue
        keep += name
        total += bytes
    }
    return keep
}
