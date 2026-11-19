package net.kdhmedia.mochil.auth

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import java.util.concurrent.atomic.AtomicBoolean

object DeviceSessionEvents {
    private val expirationActive = AtomicBoolean(false)
    private val mutableGeneration = MutableStateFlow(0L)
    val generation = mutableGeneration.asStateFlow()

    fun expired() {
        if (expirationActive.compareAndSet(false, true)) mutableGeneration.update { it + 1 }
    }

    fun connected() { expirationActive.set(false) }
}
