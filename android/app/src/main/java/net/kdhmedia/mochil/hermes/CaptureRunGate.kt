package net.kdhmedia.mochil.hermes

import java.util.concurrent.atomic.AtomicInteger

/** Invalidates blocked capture workers across stop/restart so they cannot affect a newer AudioRecord session. */
internal class CaptureRunGate {
    private val generation = AtomicInteger()

    fun start(): Int = generation.incrementAndGet()

    fun stop() {
        generation.incrementAndGet()
    }

    fun isCurrent(run: Int): Boolean = generation.get() == run
}

internal inline fun <T> startCaptureOrCleanup(start: () -> T, cleanup: () -> Unit): T =
    try {
        start()
    } catch (failure: Throwable) {
        runCatching(cleanup)
        throw failure
    }
