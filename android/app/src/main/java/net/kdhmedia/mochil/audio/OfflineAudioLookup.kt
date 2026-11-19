package net.kdhmedia.mochil.audio

/** Match cheap metadata before asking local storage or a document provider to open anything. */
internal fun <T> resolveOfflineAudioLocation(
    entries: Iterable<T>,
    requested: AudioTrack,
    trackOf: (T) -> AudioTrack?,
    resolve: (T) -> String?,
): String? {
    for (entry in entries) {
        val stored = trackOf(entry) ?: continue
        if (!sameOfflineAudioIdentity(stored, requested)) continue
        resolve(entry)?.let { return it }
    }
    return null
}
