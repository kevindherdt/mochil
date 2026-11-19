package net.kdhmedia.mochil.wiki

/** Page identity alone cannot distinguish leaving and reopening the same page. UI-thread owned. */
internal class WikiPageSession {
    private var token = Any()
    fun capture(): Any = token
    fun reset() { token = Any() }
    fun isCurrent(candidate: Any): Boolean = candidate === token
}
