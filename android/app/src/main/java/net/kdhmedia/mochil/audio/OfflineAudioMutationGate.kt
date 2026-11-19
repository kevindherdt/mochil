package net.kdhmedia.mochil.audio

/** Serializes file mutations without holding the index monitor during network transfers. */
internal class OfflineAudioMutationGate(private val indexLock: Any) {
    private val mutations = Any()

    fun <T> transfer(fetch: () -> T, publish: (T) -> Unit) = synchronized(mutations) {
        val downloaded = fetch()
        synchronized(indexLock) { publish(downloaded) }
    }

    fun <T> mutate(action: () -> T): T = synchronized(mutations) {
        synchronized(indexLock) { action() }
    }
}
