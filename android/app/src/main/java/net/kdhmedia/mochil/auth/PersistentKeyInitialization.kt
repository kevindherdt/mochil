package net.kdhmedia.mochil.auth

private val persistentKeyInitializationLock = Any()

/** Check/create is atomic across store instances; encryption and disk writes stay outside this lock. */
internal fun <T : Any> getOrCreatePersistentKey(load: () -> T?, create: () -> T): T =
    synchronized(persistentKeyInitializationLock) { load() ?: create() }
